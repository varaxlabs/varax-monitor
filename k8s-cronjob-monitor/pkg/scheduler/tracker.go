package scheduler

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"

	"github.com/kubeshield/k8s-cronjob-monitor/pkg/metrics"
)

// CronJobGetter interface for getting all CronJobs
type CronJobGetter interface {
	GetAllCronJobs() []*batchv1.CronJob
}

// Tracker monitors CronJob schedules and detects missed executions
type Tracker struct {
	metrics       *metrics.Collector
	cronJobGetter CronJobGetter
	parser        cron.Parser
	logger        *zap.Logger
	gracePeriod   time.Duration
	checkInterval time.Duration
}

// NewTracker creates a new schedule tracker
func NewTracker(
	metricsCollector *metrics.Collector,
	cronJobGetter CronJobGetter,
	logger *zap.Logger,
) *Tracker {
	return &Tracker{
		metrics:       metricsCollector,
		cronJobGetter: cronJobGetter,
		// Standard cron parser with seconds support (optional)
		parser: cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		),
		logger:        logger,
		gracePeriod:   5 * time.Minute, // Allow 5 minutes grace period
		checkInterval: 1 * time.Minute, // Check every minute
	}
}

// Start begins the schedule tracking loop
func (t *Tracker) Start(ctx context.Context) {
	t.logger.Info("Starting schedule tracker",
		zap.Duration("grace_period", t.gracePeriod),
		zap.Duration("check_interval", t.checkInterval),
	)

	// Run immediately on start
	t.checkAllSchedules()

	ticker := time.NewTicker(t.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.checkAllSchedules()
		case <-ctx.Done():
			t.logger.Info("Schedule tracker stopped")
			return
		}
	}
}

// checkAllSchedules checks all CronJobs for missed schedules
func (t *Tracker) checkAllSchedules() {
	cronJobs := t.cronJobGetter.GetAllCronJobs()
	if cronJobs == nil {
		return
	}

	now := time.Now()

	for _, cronJob := range cronJobs {
		t.checkCronJobSchedule(cronJob, now)
	}
}

// checkCronJobSchedule checks a single CronJob for missed schedules
func (t *Tracker) checkCronJobSchedule(cronJob *batchv1.CronJob, now time.Time) {
	// Skip suspended CronJobs
	if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
		return
	}

	// Parse the cron schedule
	schedule, err := t.parser.Parse(cronJob.Spec.Schedule)
	if err != nil {
		t.logger.Debug("Failed to parse cron schedule",
			zap.String("namespace", cronJob.Namespace),
			zap.String("name", cronJob.Name),
			zap.String("schedule", cronJob.Spec.Schedule),
			zap.Error(err),
		)
		return
	}

	// Calculate next run time and update metric
	var nextRun time.Time
	if cronJob.Status.LastScheduleTime != nil {
		nextRun = schedule.Next(cronJob.Status.LastScheduleTime.Time)
	} else {
		// Never run before, use creation time as base
		nextRun = schedule.Next(cronJob.CreationTimestamp.Time)
	}

	// Update next schedule timestamp metric
	t.metrics.UpdateNextSchedule(cronJob.Namespace, cronJob.Name, nextRun.Unix())

	// Check for missed schedule
	// A schedule is missed if:
	// 1. The expected run time + grace period has passed
	// 2. There are no active jobs running
	// 3. The last schedule time hasn't been updated

	if now.After(nextRun.Add(t.gracePeriod)) {
		// Check if there are active jobs
		hasActiveJobs := cronJob.Status.Active != nil && len(cronJob.Status.Active) > 0

		if !hasActiveJobs {
			// Check if this is a genuinely missed schedule
			// (not just a stale check from before a job ran)
			if cronJob.Status.LastScheduleTime == nil || cronJob.Status.LastScheduleTime.Time.Before(nextRun) {
				t.logger.Warn("Missed schedule detected",
					zap.String("namespace", cronJob.Namespace),
					zap.String("name", cronJob.Name),
					zap.String("schedule", cronJob.Spec.Schedule),
					zap.Time("expected_run", nextRun),
					zap.Duration("overdue_by", now.Sub(nextRun)),
				)

				t.metrics.RecordMissedSchedule(cronJob.Namespace, cronJob.Name)
			}
		}
	}
}
