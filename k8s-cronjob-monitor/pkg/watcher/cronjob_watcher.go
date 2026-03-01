package watcher

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeshield/k8s-cronjob-monitor/pkg/metrics"
)

// CronJobWatcher watches CronJob resources and updates metrics
type CronJobWatcher struct {
	clientset *kubernetes.Clientset
	metrics   *metrics.Collector
	logger    *zap.Logger
	informer  cache.SharedIndexInformer
}

// NewCronJobWatcher creates a new CronJob watcher with auto-detection of K8s config
func NewCronJobWatcher(metricsCollector *metrics.Collector, logger *zap.Logger) (*CronJobWatcher, error) {
	config, err := getKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	logger.Info("Kubernetes client initialized successfully")

	return &CronJobWatcher{
		clientset: clientset,
		metrics:   metricsCollector,
		logger:    logger,
	}, nil
}

// GetClientset returns the Kubernetes clientset for use by other watchers
func (w *CronJobWatcher) GetClientset() *kubernetes.Clientset {
	return w.clientset
}

// getKubernetesConfig attempts to get K8s config with auto-detection
// First tries in-cluster config, then falls back to local kubeconfig
func getKubernetesConfig() (*rest.Config, error) {
	// Try in-cluster config first (when running as pod in K8s)
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to local kubeconfig (for development)
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err = kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	return config, nil
}

// Start begins watching CronJobs using SharedInformer
func (w *CronJobWatcher) Start(ctx context.Context) error {
	// Create informer factory (watches all namespaces, resync every 30 seconds)
	factory := informers.NewSharedInformerFactory(w.clientset, 30*time.Second)

	// Get CronJob informer
	w.informer = factory.Batch().V1().CronJobs().Informer()

	// Add event handlers
	_, err := w.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cronJob := obj.(*batchv1.CronJob)
			w.logger.Info("CronJob discovered",
				zap.String("namespace", cronJob.Namespace),
				zap.String("name", cronJob.Name),
				zap.String("schedule", cronJob.Spec.Schedule),
			)

			// Initialize metrics for this CronJob
			w.metrics.InitializeCronJob(cronJob.Namespace, cronJob.Name)

			// Update active jobs count
			if cronJob.Status.Active != nil {
				w.metrics.UpdateActiveJobs(cronJob.Namespace, cronJob.Name, len(cronJob.Status.Active))
			}
		},

		UpdateFunc: func(oldObj, newObj interface{}) {
			cronJob := newObj.(*batchv1.CronJob)

			// Update active jobs count on changes
			if cronJob.Status.Active != nil {
				w.metrics.UpdateActiveJobs(cronJob.Namespace, cronJob.Name, len(cronJob.Status.Active))
			} else {
				w.metrics.UpdateActiveJobs(cronJob.Namespace, cronJob.Name, 0)
			}
		},

		DeleteFunc: func(obj interface{}) {
			cronJob := obj.(*batchv1.CronJob)
			w.logger.Info("CronJob deleted",
				zap.String("namespace", cronJob.Namespace),
				zap.String("name", cronJob.Name),
			)

			// Clean up metrics
			w.metrics.RemoveCronJob(cronJob.Namespace, cronJob.Name)
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add event handler: %w", err)
	}

	// Start informer
	factory.Start(ctx.Done())

	// Wait for cache sync
	w.logger.Info("Waiting for CronJob informer cache to sync...")
	if !cache.WaitForCacheSync(ctx.Done(), w.informer.HasSynced) {
		return fmt.Errorf("failed to sync CronJob cache")
	}
	w.logger.Info("CronJob informer cache synced")

	return nil
}

// GetAllCronJobs returns all CronJobs currently in the cache
func (w *CronJobWatcher) GetAllCronJobs() []*batchv1.CronJob {
	if w.informer == nil {
		return nil
	}

	items := w.informer.GetStore().List()
	cronJobs := make([]*batchv1.CronJob, 0, len(items))

	for _, item := range items {
		if cronJob, ok := item.(*batchv1.CronJob); ok {
			cronJobs = append(cronJobs, cronJob)
		}
	}

	return cronJobs
}
