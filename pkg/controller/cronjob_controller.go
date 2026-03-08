package controller

import (
	"context"
	"fmt"
	"sort"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/varaxlabs/onax/pkg/metrics"
	"github.com/varaxlabs/onax/pkg/models"
)

const jobOwnerField = ".metadata.ownerReferences.cronjob"

// CronJobReconciler reconciles CronJob objects and updates Prometheus metrics.
type CronJobReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Collector *metrics.Collector
}

// Reconcile fetches a CronJob, lists its owned Jobs, computes status, and updates metrics.
func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cronJob batchv1.CronJob
	if err := r.Get(ctx, req.NamespacedName, &cronJob); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("CronJob deleted, removing metrics", "cronjob", req.NamespacedName)
			r.Collector.RemoveCronJob(req.Namespace, req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// List only Jobs owned by this CronJob using the field index
	var jobList batchv1.JobList
	if err := r.List(ctx, &jobList,
		client.InNamespace(req.Namespace),
		client.MatchingFields{jobOwnerField: req.Name},
	); err != nil {
		return ctrl.Result{}, err
	}

	// Sort most-recent-first
	sort.Slice(jobList.Items, func(i, j int) bool {
		return models.JobStartTime(&jobList.Items[i]).After(models.JobStartTime(&jobList.Items[j]))
	})

	// Compute status
	status := models.ComputeStatus(&cronJob, jobList.Items)

	// Update metrics
	r.Collector.UpdateAll(status)

	// Requeue after 60s to keep metrics fresh
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// SetupWithManager registers the field indexer and reconciler with the manager.
func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Index Jobs by the name of the owning CronJob so we can filter
	// with MatchingFields instead of listing all Jobs in the namespace.
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&batchv1.Job{},
		jobOwnerField,
		func(obj client.Object) []string {
			job, ok := obj.(*batchv1.Job)
			if !ok {
				return nil
			}
			owner, _ := models.GetCronJobOwner(job)
			if owner == "" {
				return nil
			}
			return []string{owner}
		},
	); err != nil {
		return fmt.Errorf("failed to create field index for Jobs: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.CronJob{}).
		Watches(&batchv1.Job{}, handler.EnqueueRequestsFromMapFunc(r.mapJobToCronJob)).
		Complete(r)
}

// mapJobToCronJob maps a Job event to a reconcile request for its owning CronJob.
func (r *CronJobReconciler) mapJobToCronJob(ctx context.Context, obj client.Object) []reconcile.Request {
	job, ok := obj.(*batchv1.Job)
	if !ok {
		return nil
	}

	name, _ := models.GetCronJobOwner(job)
	if name == "" {
		return nil
	}

	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Namespace: job.Namespace,
			Name:      name,
		}},
	}
}
