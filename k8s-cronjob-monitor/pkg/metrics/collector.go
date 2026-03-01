package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector holds all Prometheus metrics for CronJob monitoring
type Collector struct {
	// Gauges
	executionStatus   *prometheus.GaugeVec
	executionDuration *prometheus.GaugeVec
	lastSuccessTime   *prometheus.GaugeVec
	lastFailureTime   *prometheus.GaugeVec
	nextScheduleTime  *prometheus.GaugeVec
	activeJobs        *prometheus.GaugeVec
	successRate       *prometheus.GaugeVec
	scheduleDelay     *prometheus.GaugeVec

	// Counters
	executionsTotal *prometheus.CounterVec
	missedSchedules *prometheus.CounterVec

	// Internal tracking for success rate calculation
	successCounts map[string]int64
	failureCounts map[string]int64
	countsMu      sync.RWMutex
}

// NewCollector creates a new metrics collector with all Prometheus metrics registered
func NewCollector() *Collector {
	c := &Collector{
		successCounts: make(map[string]int64),
		failureCounts: make(map[string]int64),
	}

	c.executionStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_execution_status",
			Help: "Current status of last CronJob execution (1=success, 0=failed, -1=unknown)",
		},
		[]string{"namespace", "cronjob"},
	)

	c.executionDuration = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_execution_duration_seconds",
			Help: "Duration of last CronJob execution in seconds",
		},
		[]string{"namespace", "cronjob"},
	)

	c.executionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cronjob_monitor_executions_total",
			Help: "Total number of CronJob executions",
		},
		[]string{"namespace", "cronjob", "status"},
	)

	c.lastSuccessTime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_last_success_timestamp",
			Help: "Timestamp of last successful execution",
		},
		[]string{"namespace", "cronjob"},
	)

	c.lastFailureTime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_last_failure_timestamp",
			Help: "Timestamp of last failed execution",
		},
		[]string{"namespace", "cronjob"},
	)

	c.nextScheduleTime = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_next_schedule_timestamp",
			Help: "Expected timestamp of next execution",
		},
		[]string{"namespace", "cronjob"},
	)

	c.missedSchedules = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cronjob_monitor_missed_schedules_total",
			Help: "Total number of missed schedules",
		},
		[]string{"namespace", "cronjob"},
	)

	c.activeJobs = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_active_jobs",
			Help: "Current number of active Job pods",
		},
		[]string{"namespace", "cronjob"},
	)

	c.successRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_success_rate",
			Help: "Success rate (successful / total) over last 24 hours",
		},
		[]string{"namespace", "cronjob"},
	)

	c.scheduleDelay = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cronjob_monitor_schedule_delay_seconds",
			Help: "Difference between scheduled and actual start time",
		},
		[]string{"namespace", "cronjob"},
	)

	return c
}

// InitializeCronJob sets initial metrics when a CronJob is discovered
func (c *Collector) InitializeCronJob(namespace, name string) {
	// Set unknown status initially
	c.executionStatus.WithLabelValues(namespace, name).Set(-1)
	c.activeJobs.WithLabelValues(namespace, name).Set(0)

	// Initialize counters (they need to exist for queries to work)
	c.executionsTotal.WithLabelValues(namespace, name, "success")
	c.executionsTotal.WithLabelValues(namespace, name, "failed")
	c.missedSchedules.WithLabelValues(namespace, name)
}

// RemoveCronJob cleans up metrics when a CronJob is deleted
func (c *Collector) RemoveCronJob(namespace, name string) {
	labels := prometheus.Labels{"namespace": namespace, "cronjob": name}

	c.executionStatus.Delete(labels)
	c.executionDuration.Delete(labels)
	c.lastSuccessTime.Delete(labels)
	c.lastFailureTime.Delete(labels)
	c.nextScheduleTime.Delete(labels)
	c.activeJobs.Delete(labels)
	c.successRate.Delete(labels)
	c.scheduleDelay.Delete(labels)

	// Delete counter labels
	c.executionsTotal.Delete(prometheus.Labels{"namespace": namespace, "cronjob": name, "status": "success"})
	c.executionsTotal.Delete(prometheus.Labels{"namespace": namespace, "cronjob": name, "status": "failed"})
	c.missedSchedules.Delete(labels)

	// Clean up internal tracking
	c.countsMu.Lock()
	key := namespace + "/" + name
	delete(c.successCounts, key)
	delete(c.failureCounts, key)
	c.countsMu.Unlock()
}

// RecordJobStart records when a job starts
func (c *Collector) RecordJobStart(namespace, cronjob string, activeCount int) {
	c.activeJobs.WithLabelValues(namespace, cronjob).Set(float64(activeCount))
}

// RecordSuccess records a successful job execution
func (c *Collector) RecordSuccess(namespace, cronjob string, duration float64) {
	c.executionStatus.WithLabelValues(namespace, cronjob).Set(1)
	c.executionDuration.WithLabelValues(namespace, cronjob).Set(duration)
	c.executionsTotal.WithLabelValues(namespace, cronjob, "success").Inc()
	c.lastSuccessTime.WithLabelValues(namespace, cronjob).Set(float64(time.Now().Unix()))

	// Update success rate tracking
	c.countsMu.Lock()
	key := namespace + "/" + cronjob
	c.successCounts[key]++
	c.updateSuccessRate(namespace, cronjob)
	c.countsMu.Unlock()
}

// RecordFailure records a failed job execution
func (c *Collector) RecordFailure(namespace, cronjob string, duration float64) {
	c.executionStatus.WithLabelValues(namespace, cronjob).Set(0)
	c.executionDuration.WithLabelValues(namespace, cronjob).Set(duration)
	c.executionsTotal.WithLabelValues(namespace, cronjob, "failed").Inc()
	c.lastFailureTime.WithLabelValues(namespace, cronjob).Set(float64(time.Now().Unix()))

	// Update success rate tracking
	c.countsMu.Lock()
	key := namespace + "/" + cronjob
	c.failureCounts[key]++
	c.updateSuccessRate(namespace, cronjob)
	c.countsMu.Unlock()
}

// RecordMissedSchedule increments the missed schedule counter
func (c *Collector) RecordMissedSchedule(namespace, cronjob string) {
	c.missedSchedules.WithLabelValues(namespace, cronjob).Inc()
}

// UpdateNextSchedule updates the next expected run timestamp
func (c *Collector) UpdateNextSchedule(namespace, cronjob string, timestamp int64) {
	c.nextScheduleTime.WithLabelValues(namespace, cronjob).Set(float64(timestamp))
}

// UpdateActiveJobs sets the current number of active jobs
func (c *Collector) UpdateActiveJobs(namespace, cronjob string, count int) {
	c.activeJobs.WithLabelValues(namespace, cronjob).Set(float64(count))
}

// RecordScheduleDelay records the delay between scheduled and actual start time
func (c *Collector) RecordScheduleDelay(namespace, cronjob string, delaySeconds float64) {
	c.scheduleDelay.WithLabelValues(namespace, cronjob).Set(delaySeconds)
}

// updateSuccessRate recalculates the success rate (must be called with lock held)
func (c *Collector) updateSuccessRate(namespace, cronjob string) {
	key := namespace + "/" + cronjob
	successes := c.successCounts[key]
	failures := c.failureCounts[key]
	total := successes + failures

	if total == 0 {
		c.successRate.WithLabelValues(namespace, cronjob).Set(1.0) // No executions = 100% success
		return
	}

	rate := float64(successes) / float64(total)
	c.successRate.WithLabelValues(namespace, cronjob).Set(rate)
}
