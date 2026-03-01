package watcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeshield/k8s-cronjob-monitor/pkg/metrics"
)

// JobWatcher watches Job resources created by CronJobs and updates metrics
type JobWatcher struct {
	clientset     *kubernetes.Clientset
	metrics       *metrics.Collector
	logger        *zap.Logger
	jobStartTimes map[string]time.Time
	processedJobs map[string]bool // Track which jobs we've already processed completion for
	mu            sync.RWMutex
}

// NewJobWatcher creates a new Job watcher
func NewJobWatcher(clientset *kubernetes.Clientset, metricsCollector *metrics.Collector, logger *zap.Logger) *JobWatcher {
	return &JobWatcher{
		clientset:     clientset,
		metrics:       metricsCollector,
		logger:        logger,
		jobStartTimes: make(map[string]time.Time),
		processedJobs: make(map[string]bool),
	}
}

// Start begins watching Jobs using SharedInformer
func (w *JobWatcher) Start(ctx context.Context) error {
	// Create informer factory
	factory := informers.NewSharedInformerFactory(w.clientset, 30*time.Second)

	// Get Job informer
	informer := factory.Batch().V1().Jobs().Informer()

	// Add event handlers
	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			job := obj.(*batchv1.Job)
			w.handleJobEvent(job)
		},

		UpdateFunc: func(oldObj, newObj interface{}) {
			job := newObj.(*batchv1.Job)
			w.handleJobEvent(job)
		},

		DeleteFunc: func(obj interface{}) {
			job := obj.(*batchv1.Job)
			// Clean up tracking maps
			jobKey := fmt.Sprintf("%s/%s", job.Namespace, job.Name)
			w.mu.Lock()
			delete(w.jobStartTimes, jobKey)
			delete(w.processedJobs, jobKey)
			w.mu.Unlock()
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add event handler: %w", err)
	}

	// Start informer
	factory.Start(ctx.Done())

	// Wait for cache sync
	w.logger.Info("Waiting for Job informer cache to sync...")
	if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
		return fmt.Errorf("failed to sync Job cache")
	}
	w.logger.Info("Job informer cache synced")

	return nil
}

// handleJobEvent processes a job event and updates metrics
func (w *JobWatcher) handleJobEvent(job *batchv1.Job) {
	// Only track jobs created by CronJobs
	cronJobName := getCronJobName(job)
	if cronJobName == "" {
		return
	}

	jobKey := fmt.Sprintf("%s/%s", job.Namespace, job.Name)

	// Track job start
	if job.Status.Active > 0 {
		w.mu.Lock()
		if _, exists := w.jobStartTimes[jobKey]; !exists {
			w.jobStartTimes[jobKey] = time.Now()
			w.logger.Debug("Job started",
				zap.String("namespace", job.Namespace),
				zap.String("job", job.Name),
				zap.String("cronjob", cronJobName),
			)

			// Record schedule delay if we have the start time
			if job.Status.StartTime != nil {
				// Get the CronJob to check scheduled time
				cronJob, err := w.clientset.BatchV1().CronJobs(job.Namespace).Get(
					context.Background(), cronJobName, metav1.GetOptions{})
				if err == nil && cronJob.Status.LastScheduleTime != nil {
					delay := job.Status.StartTime.Sub(cronJob.Status.LastScheduleTime.Time).Seconds()
					if delay >= 0 {
						w.metrics.RecordScheduleDelay(job.Namespace, cronJobName, delay)
					}
				}
			}
		}
		w.mu.Unlock()
	}

	// Check if job completed (success or failure)
	if isJobComplete(job) {
		w.mu.Lock()
		// Only process completion once
		if w.processedJobs[jobKey] {
			w.mu.Unlock()
			return
		}
		w.processedJobs[jobKey] = true

		startTime := w.jobStartTimes[jobKey]
		w.mu.Unlock()

		// Calculate duration
		duration := float64(0)
		if job.Status.CompletionTime != nil && job.Status.StartTime != nil {
			duration = job.Status.CompletionTime.Sub(job.Status.StartTime.Time).Seconds()
		} else if !startTime.IsZero() {
			duration = time.Since(startTime).Seconds()
		}

		if isJobSuccessful(job) {
			w.logger.Info("Job completed successfully",
				zap.String("namespace", job.Namespace),
				zap.String("job", job.Name),
				zap.String("cronjob", cronJobName),
				zap.Float64("duration_seconds", duration),
			)

			w.metrics.RecordSuccess(job.Namespace, cronJobName, duration)
		} else {
			exitCode := w.getJobExitCode(job)
			errorMsg := w.getJobError(job)

			w.logger.Warn("Job failed",
				zap.String("namespace", job.Namespace),
				zap.String("job", job.Name),
				zap.String("cronjob", cronJobName),
				zap.Float64("duration_seconds", duration),
				zap.Int32p("exit_code", exitCode),
				zap.String("error", errorMsg),
			)

			w.metrics.RecordFailure(job.Namespace, cronJobName, duration)
		}

		// Update active jobs count (will be 0 after completion)
		w.metrics.UpdateActiveJobs(job.Namespace, cronJobName, 0)
	}
}

// getCronJobName extracts the CronJob name from a Job's owner references
func getCronJobName(job *batchv1.Job) string {
	for _, owner := range job.OwnerReferences {
		if owner.Kind == "CronJob" {
			return owner.Name
		}
	}
	return ""
}

// isJobComplete checks if a job has completed (success or failure)
func isJobComplete(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete || condition.Type == batchv1.JobFailed {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// isJobSuccessful checks if a completed job was successful
func isJobSuccessful(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// getJobExitCode attempts to get the exit code from a job's pod
func (w *JobWatcher) getJobExitCode(job *batchv1.Job) *int32 {
	pods, err := w.clientset.CoreV1().Pods(job.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
	})
	if err != nil || len(pods.Items) == 0 {
		return nil
	}

	// Get the first pod's container status
	pod := pods.Items[0]
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Terminated != nil {
			exitCode := containerStatus.State.Terminated.ExitCode
			return &exitCode
		}
	}

	return nil
}

// getJobError attempts to get error message from a failed job
func (w *JobWatcher) getJobError(job *batchv1.Job) string {
	pods, err := w.clientset.CoreV1().Pods(job.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
	})
	if err != nil || len(pods.Items) == 0 {
		return ""
	}

	// Get the first pod's error
	pod := pods.Items[0]
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.Reason != "" {
			return containerStatus.State.Terminated.Reason + ": " + containerStatus.State.Terminated.Message
		}
		if containerStatus.State.Waiting != nil && containerStatus.State.Waiting.Reason != "" {
			return containerStatus.State.Waiting.Reason + ": " + containerStatus.State.Waiting.Message
		}
	}

	return ""
}
