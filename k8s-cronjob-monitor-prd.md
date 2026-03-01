# Kubernetes CronJob Monitor — Product Requirements Document

**Version**: 2.0
**Last Updated**: February 2026
**Status**: Active Development
**Build Timeline**: 2–3 weeks
**License**: Apache 2.0 (100% Free and Open Source)

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [The Problem](#the-problem)
3. [The Solution](#the-solution)
4. [Strategic Purpose](#strategic-purpose)
5. [System Architecture](#system-architecture)
6. [Technology Stack](#technology-stack)
7. [Repository Structure](#repository-structure)
8. [Core Logic](#core-logic)
9. [Prometheus Metrics](#prometheus-metrics)
10. [Grafana Dashboards](#grafana-dashboards)
11. [Prometheus Alert Rules](#prometheus-alert-rules)
12. [Helm Chart](#helm-chart)
13. [RBAC & Security](#rbac--security)
14. [Installation Guide](#installation-guide)
15. [Development Setup](#development-setup)
16. [Testing Strategy](#testing-strategy)
17. [Development Phases](#development-phases)
18. [Launch Plan](#launch-plan)
19. [Success Metrics](#success-metrics)
20. [Competitive Positioning](#competitive-positioning)

---

## Executive Summary

**Product Name**: k8s-cronjob-monitor
**Tagline**: "Dead-simple CronJob monitoring for Kubernetes with Prometheus and Grafana"
**Business Model**: 100% Free and Open Source (Apache 2.0)
**Strategic Purpose**: Brand building and lead generation for KubeShield Compliance Platform
**Time to Build**: 2–3 weeks
**Target Adoption**: 1,000+ GitHub stars, 5,000+ installations in 12 months

k8s-cronjob-monitor is a lightweight Kubernetes operator that auto-discovers all CronJobs in a cluster, tracks their execution status, and exports Prometheus metrics with pre-built Grafana dashboards. It installs in one command, requires zero configuration, and data never leaves the cluster.

---

## The Problem

Kubernetes CronJob monitoring has two bad options:

### Option 1: SaaS Tools (Cronitor, Healthchecks.io, Dead Man's Snitch)
- Cost $21–$449/month
- Require manual per-job configuration (no auto-discovery)
- Heartbeat-based, not Kubernetes-native
- Data leaves the cluster (compliance concern)
- Complex setup that defeats the purpose of "easy monitoring"

### Option 2: DIY Prometheus Setup
- Requires combining 4+ disparate metrics from kube-state-metrics
- Complex PromQL queries that are easy to misconfigure
- Manual Grafana dashboard creation
- Hours of debugging YAML indentation and scrape configs
- No built-in alert rules for CronJob-specific scenarios

**Result**: Most teams either pay for expensive SaaS tools that don't fit Kubernetes well, or live with unreliable CronJobs that fail silently.

---

## The Solution

k8s-cronjob-monitor is a lightweight Kubernetes operator that:

- **Auto-discovers** all CronJobs in the cluster (zero configuration)
- **Tracks** execution status, duration, success/failure rates, and missed schedules
- **Exports** native Prometheus metrics for alerting and visualization
- **Includes** pre-built Grafana dashboards ready for import
- **Provides** pre-configured Prometheus alert rules
- **Runs entirely** in the customer's cluster (data never leaves)
- **Installs** with one Helm command in under 60 seconds
- **Uses** <50MB memory, <0.05 CPU cores

### Value Proposition

**For DevOps Teams**: One-command install, zero configuration, works with existing Prometheus/Grafana stack. No new SaaS subscriptions.

**For Budget-Conscious Teams**: Completely free. No per-job pricing. No hidden costs. Unlimited CronJobs.

**For Privacy-Focused Teams**: All data stays in cluster. No external dependencies. Full control over data retention.

---

## Strategic Purpose

This project serves multiple purposes within the KubeShield product suite:

1. **Brand Building**: Establish KubeShield as Kubernetes experts in the community
2. **Lead Generation**: Drive users toward the KubeShield Compliance Platform (the revenue product)
3. **Learning Investment**: Practice building Kubernetes operators before the more complex compliance platform
4. **Community Building**: Create an engaged user base that trusts the KubeShield brand
5. **Cross-Promotion Channel**: README, docs, and in-operator output all reference KubeShield Compliance

### Cross-Promotion Placement

**In README.md:**
```markdown
## 🔒 Need Kubernetes Compliance Automation?

Check out [KubeShield Compliance Platform](https://kubeshield.io) by the same team:
- Automated SOC2/HIPAA/PCI-DSS compliance for Kubernetes
- Auto-enable audit logging across EKS/AKS/GKE
- Generate audit-ready evidence reports
- Same Kubernetes-native approach you love
```

**In Helm NOTES.txt (shown after install):**
```
🎉 k8s-cronjob-monitor is running!

Need compliance automation? Try KubeShield → https://kubeshield.io
```

---

## System Architecture

```
┌──────────────────────────────────────────────────────────────┐
│              Kubernetes Cluster (User's)                       │
│                                                                │
│  ┌──────────────────────────────────────────────────────┐    │
│  │  CronJobs (Auto-discovered)                           │    │
│  │  ├── backup-job (every night at 2am)                 │    │
│  │  ├── report-generator (daily at 9am)                 │    │
│  │  └── cleanup-job (weekly)                             │    │
│  └────────────┬──────────────────────────────────────────┘    │
│               │ Watch (Informers)                              │
│               │                                                │
│  ┌────────────▼──────────────────────────────────────────┐    │
│  │  k8s-cronjob-monitor Operator (Go)                     │    │
│  │                                                         │    │
│  │  ┌───────────────────────────────────────────────┐    │    │
│  │  │ CronJob Watcher                                │    │    │
│  │  │ • List all CronJobs via K8s API               │    │    │
│  │  │ • Watch for Job executions                    │    │    │
│  │  │ • Track success/failure/duration              │    │    │
│  │  │ • Detect missed schedules                     │    │    │
│  │  └───────────────────────────────────────────────┘    │    │
│  │  ┌───────────────────────────────────────────────┐    │    │
│  │  │ Metrics Exporter                               │    │    │
│  │  │ • Expose Prometheus endpoint :8080/metrics    │    │    │
│  │  │ • Update metrics on every Job event           │    │    │
│  │  └───────────────────────────────────────────────┘    │    │
│  └────────────┬──────────────────────────────────────────┘    │
│               │ /metrics endpoint                              │
│               │                                                │
│  ┌────────────▼──────────────────────────────────────────┐    │
│  │  Prometheus (User's existing)                          │    │
│  │  • Scrapes /metrics every 15–30s                      │    │
│  │  • Stores time-series data                            │    │
│  │  • Evaluates alert rules                              │    │
│  └────────────┬──────────────────────────────────────────┘    │
│               │                                                │
│               ├──► AlertManager → Slack/Email/PagerDuty        │
│               │                                                │
│  ┌────────────▼──────────────────────────────────────────┐    │
│  │  Grafana (User's existing)                             │    │
│  │  • Pre-built dashboards (import JSON)                 │    │
│  │  • CronJob overview + per-job drilldown               │    │
│  │  • Team sharing                                        │    │
│  └────────────────────────────────────────────────────────┘    │
└────────────────────────────────────────────────────────────────┘
```

**Key architectural decisions:**
- **No backend API** — data stays in cluster, exported as Prometheus metrics
- **No frontend** — uses customer's existing Grafana with pre-built dashboards
- **No database** — Prometheus handles time-series storage and retention
- **No authentication** — uses Kubernetes RBAC (ServiceAccount with read-only access)
- **Single binary** — one Go operator, one container, one Deployment

---

## Technology Stack

### Operator (Go 1.22+)
| Dependency | Version | Purpose |
|-----------|---------|---------|
| **Kubebuilder** | 3.14+ | Operator framework / scaffolding |
| **controller-runtime** | v0.17+ | Reconciliation loops, caching |
| **client-go** | v0.30+ | Kubernetes API client |
| **prometheus/client_golang** | v1.19+ | Metrics exporter |
| **zap** | v1.27+ | Structured logging |

### Monitoring Stack (User's Existing)
| Tool | Minimum Version | Purpose |
|------|----------------|---------|
| **Prometheus** | Any | Metrics scraping and storage |
| **Grafana** | 8.0+ | Dashboard visualization |
| **AlertManager** | Any (optional) | Alert routing |

### Development Tools
| Tool | Version | Purpose |
|------|---------|---------|
| **kind** | v0.22+ | Local Kubernetes clusters for testing |
| **Docker** | 24.0+ | Container builds |
| **Helm** | 3.0+ | Package management |

---

## Repository Structure

```
k8s-cronjob-monitor/
├── cmd/
│   └── operator/
│       └── main.go                        # Operator entry point
├── pkg/
│   ├── controller/
│   │   └── cronjob_controller.go          # Main reconciler
│   ├── watcher/
│   │   ├── cronjob_watcher.go             # Watch CronJob resources
│   │   ├── job_watcher.go                 # Watch Job executions
│   │   └── schedule_tracker.go            # Detect missed schedules
│   ├── metrics/
│   │   ├── exporter.go                    # Prometheus exporter setup
│   │   └── collector.go                   # Metric collection logic
│   └── models/
│       └── cronjob_status.go              # Internal data models
├── config/
│   ├── rbac/
│   │   ├── role.yaml                      # ClusterRole (read-only)
│   │   └── rolebinding.yaml               # ClusterRoleBinding
│   ├── samples/
│   │   └── test-cronjob.yaml              # Example CronJob for testing
│   └── crd/                               # (None — uses built-in K8s resources)
├── deploy/
│   ├── helm/
│   │   └── k8s-cronjob-monitor/
│   │       ├── Chart.yaml
│   │       ├── values.yaml
│   │       └── templates/
│   │           ├── deployment.yaml
│   │           ├── service.yaml
│   │           ├── servicemonitor.yaml    # For Prometheus Operator users
│   │           ├── rbac.yaml
│   │           └── NOTES.txt
│   └── manifests/
│       └── operator.yaml                  # Raw YAML install (no Helm)
├── dashboards/
│   ├── cronjob-overview.json              # Main overview dashboard
│   ├── cronjob-details.json               # Per-job drilldown dashboard
│   └── README.md                          # Import instructions
├── alerts/
│   └── cronjob-alerts.yaml                # Prometheus alert rules
├── docs/
│   ├── installation.md
│   ├── configuration.md
│   ├── metrics.md                         # Full metric reference
│   ├── troubleshooting.md
│   └── architecture.md
├── examples/
│   ├── basic-cronjob.yaml                 # Simple test CronJob
│   ├── with-slack-alerts.yaml             # AlertManager Slack config
│   └── with-pagerduty.yaml                # AlertManager PagerDuty config
├── hack/
│   └── test-setup.sh                      # Local testing with kind
├── .github/
│   ├── workflows/
│   │   ├── ci.yaml                        # Build, lint, test
│   │   ├── release.yaml                   # Automated releases + Helm publish
│   │   └── docker-build.yaml              # Container image builds
│   ├── ISSUE_TEMPLATE/
│   └── PULL_REQUEST_TEMPLATE.md
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
├── LICENSE                                # Apache 2.0
├── CONTRIBUTING.md
└── README.md
```

---

## Core Logic

### 1. Auto-Discovery (CronJob Watcher)

Uses Kubernetes Informers to watch all CronJob resources cluster-wide. When a CronJob is created, updated, or deleted, the operator automatically initializes, updates, or removes metrics.

```go
// pkg/watcher/cronjob_watcher.go
package watcher

import (
    "context"
    "fmt"
    "time"

    batchv1 "k8s.io/api/batch/v1"
    "k8s.io/client-go/informers"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/cache"
    "go.uber.org/zap"
)

type CronJobWatcher struct {
    clientset *kubernetes.Clientset
    metrics   *metrics.Collector
    logger    *zap.Logger
}

func (w *CronJobWatcher) Start(ctx context.Context) error {
    factory := informers.NewSharedInformerFactory(w.clientset, 30*time.Second)

    cronJobInformer := factory.Batch().V1().CronJobs().Informer()

    cronJobInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            cronJob := obj.(*batchv1.CronJob)
            w.logger.Info("Discovered CronJob",
                zap.String("namespace", cronJob.Namespace),
                zap.String("name", cronJob.Name),
                zap.String("schedule", cronJob.Spec.Schedule))
            w.metrics.InitializeCronJob(cronJob)
        },
        UpdateFunc: func(oldObj, newObj interface{}) {
            cronJob := newObj.(*batchv1.CronJob)
            w.metrics.UpdateSchedule(cronJob)
        },
        DeleteFunc: func(obj interface{}) {
            cronJob := obj.(*batchv1.CronJob)
            w.logger.Info("CronJob deleted",
                zap.String("namespace", cronJob.Namespace),
                zap.String("name", cronJob.Name))
            w.metrics.RemoveCronJob(cronJob)
        },
    })

    factory.Start(ctx.Done())

    if !cache.WaitForCacheSync(ctx.Done(), cronJobInformer.HasSynced) {
        return fmt.Errorf("failed to sync CronJob cache")
    }

    return nil
}
```

### 2. Execution Tracking (Job Watcher)

Watches Job resources owned by CronJobs. Tracks start time, completion, failure, exit codes, and duration.

```go
// pkg/watcher/job_watcher.go
package watcher

import (
    batchv1 "k8s.io/api/batch/v1"
    corev1 "k8s.io/api/core/v1"
)

type JobWatcher struct {
    clientset *kubernetes.Clientset
    metrics   *metrics.Collector
    logger    *zap.Logger
}

func (w *JobWatcher) Start(ctx context.Context) error {
    factory := informers.NewSharedInformerFactory(w.clientset, 30*time.Second)

    jobInformer := factory.Batch().V1().Jobs().Informer()

    jobInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            job := obj.(*batchv1.Job)
            // Only track Jobs owned by CronJobs
            if owner := getOwnerCronJob(job); owner != "" {
                w.metrics.RecordJobStarted(job.Namespace, owner)
            }
        },
        UpdateFunc: func(oldObj, newObj interface{}) {
            job := newObj.(*batchv1.Job)
            if owner := getOwnerCronJob(job); owner != "" {
                // Check completion conditions
                for _, condition := range job.Status.Conditions {
                    switch condition.Type {
                    case batchv1.JobComplete:
                        if condition.Status == corev1.ConditionTrue {
                            duration := condition.LastTransitionTime.Sub(
                                job.Status.StartTime.Time).Seconds()
                            w.metrics.RecordJobCompleted(
                                job.Namespace, owner, duration)
                        }
                    case batchv1.JobFailed:
                        if condition.Status == corev1.ConditionTrue {
                            w.metrics.RecordJobFailed(
                                job.Namespace, owner, condition.Reason)
                        }
                    }
                }
            }
        },
    })

    factory.Start(ctx.Done())
    return nil
}

// getOwnerCronJob returns the CronJob name that owns this Job, or empty string
func getOwnerCronJob(job *batchv1.Job) string {
    for _, ref := range job.OwnerReferences {
        if ref.Kind == "CronJob" {
            return ref.Name
        }
    }
    return ""
}
```

### 3. Missed Schedule Detection

Compares expected next run time with actual Job creation timestamps to detect CronJobs that failed to fire on time.

```go
// pkg/watcher/schedule_tracker.go
package watcher

import (
    "time"

    "github.com/robfig/cron/v3"
)

type ScheduleTracker struct {
    metrics  *metrics.Collector
    parser   cron.Parser
    logger   *zap.Logger
}

// CheckMissedSchedules runs periodically to detect CronJobs that didn't fire
func (t *ScheduleTracker) CheckMissedSchedules(
    ctx context.Context, cronJobs []CronJobInfo) {

    for _, cj := range cronJobs {
        sched, err := t.parser.Parse(cj.Schedule)
        if err != nil {
            t.logger.Warn("Failed to parse schedule",
                zap.String("cronjob", cj.Name),
                zap.String("schedule", cj.Schedule))
            continue
        }

        expectedNext := sched.Next(cj.LastScheduledTime)
        gracePeriod := 5 * time.Minute

        if time.Now().After(expectedNext.Add(gracePeriod)) && !cj.HasActiveJob {
            t.metrics.RecordMissedSchedule(cj.Namespace, cj.Name)
            t.logger.Warn("Missed schedule detected",
                zap.String("namespace", cj.Namespace),
                zap.String("cronjob", cj.Name),
                zap.Time("expected", expectedNext))
        }
    }
}
```

---

## Prometheus Metrics

All metrics use the `cronjob_monitor_` prefix.

### Gauges (Current State)

```prometheus
# Current status of last CronJob execution
# Values: 1=success, 0=failed, -1=running, -2=unknown
cronjob_monitor_execution_status{namespace, cronjob}

# Duration of last CronJob execution in seconds
cronjob_monitor_execution_duration_seconds{namespace, cronjob}

# Timestamp of last successful execution (Unix)
cronjob_monitor_last_success_timestamp{namespace, cronjob}

# Timestamp of last failed execution (Unix)
cronjob_monitor_last_failure_timestamp{namespace, cronjob}

# Expected timestamp of next execution (Unix)
cronjob_monitor_next_schedule_timestamp{namespace, cronjob}

# Current number of active Job pods for this CronJob
cronjob_monitor_active_jobs{namespace, cronjob}

# Success rate over last 24h (0.0–1.0)
cronjob_monitor_success_rate{namespace, cronjob}

# Delay between scheduled and actual start time in seconds
cronjob_monitor_schedule_delay_seconds{namespace, cronjob}
```

### Counters (Cumulative)

```prometheus
# Total number of CronJob executions (partitioned by status)
cronjob_monitor_executions_total{namespace, cronjob, status="success|failed"}

# Total number of missed schedules
cronjob_monitor_missed_schedules_total{namespace, cronjob}
```

### Informational

```prometheus
# CronJob info label (schedule, suspend status)
cronjob_monitor_info{namespace, cronjob, schedule, suspended="true|false"} 1
```

---

## Grafana Dashboards

### Dashboard 1: CronJob Overview

**File**: `dashboards/cronjob-overview.json`

**Variables**: `namespace` (multi-select dropdown)

| Panel | Type | PromQL |
|-------|------|--------|
| Total CronJobs | Stat | `count(cronjob_monitor_info)` |
| Healthy CronJobs | Stat (green) | `count(cronjob_monitor_execution_status == 1)` |
| Failed CronJobs | Stat (red) | `count(cronjob_monitor_execution_status == 0)` |
| Missed Schedules (24h) | Stat (orange) | `sum(increase(cronjob_monitor_missed_schedules_total[24h]))` |
| CronJob Status Table | Table | `cronjob_monitor_execution_status` with namespace/cronjob labels |
| Execution Duration | Time series | `cronjob_monitor_execution_duration_seconds` |
| Success vs Failure Rate | Time series (stacked) | `rate(cronjob_monitor_executions_total[5m])` by status |
| Currently Running Jobs | Stat | `sum(cronjob_monitor_active_jobs)` |

### Dashboard 2: CronJob Details (Per-Job Drilldown)

**File**: `dashboards/cronjob-details.json`

**Variables**: `namespace` (single-select), `cronjob` (single-select, filtered by namespace)

| Panel | Type | PromQL |
|-------|------|--------|
| Last Execution Status | Stat (color-coded) | `cronjob_monitor_execution_status{namespace="$namespace", cronjob="$cronjob"}` |
| Last Execution Duration | Stat | `cronjob_monitor_execution_duration_seconds{...}` |
| Success Rate (24h) | Gauge | `cronjob_monitor_success_rate{...}` |
| Next Scheduled Run | Stat (timestamp) | `cronjob_monitor_next_schedule_timestamp{...}` |
| Execution History | Time series | `cronjob_monitor_execution_status{...}` |
| Duration Trend | Time series | `cronjob_monitor_execution_duration_seconds{...}` |
| Success/Failure Count | Bar gauge | `cronjob_monitor_executions_total{...}` by status |
| Schedule Delay | Time series | `cronjob_monitor_schedule_delay_seconds{...}` |

---

## Prometheus Alert Rules

**File**: `alerts/cronjob-alerts.yaml`

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cronjob-monitor-alerts
  namespace: monitoring
data:
  alerts.yaml: |
    groups:
    - name: cronjob_monitoring
      interval: 1m
      rules:

      # Alert when a CronJob fails
      - alert: CronJobFailed
        expr: cronjob_monitor_execution_status == 0
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "CronJob {{ $labels.cronjob }} failed"
          description: >
            CronJob {{ $labels.cronjob }} in namespace {{ $labels.namespace }}
            has failed its last execution.

      # Alert when a CronJob misses its schedule
      - alert: CronJobMissedSchedule
        expr: increase(cronjob_monitor_missed_schedules_total[15m]) > 0
        for: 0m
        labels:
          severity: warning
        annotations:
          summary: "CronJob {{ $labels.cronjob }} missed schedule"
          description: >
            CronJob {{ $labels.cronjob }} in namespace {{ $labels.namespace }}
            missed its scheduled execution.

      # Alert when a CronJob runs longer than expected
      - alert: CronJobSlowExecution
        expr: cronjob_monitor_execution_duration_seconds > 3600
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "CronJob {{ $labels.cronjob }} running >1 hour"
          description: >
            CronJob {{ $labels.cronjob }} in namespace {{ $labels.namespace }}
            has been running for {{ $value | humanizeDuration }}.

      # Alert when success rate drops below 90%
      - alert: CronJobLowSuccessRate
        expr: cronjob_monitor_success_rate < 0.9
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "CronJob {{ $labels.cronjob }} success rate below 90%"
          description: >
            CronJob {{ $labels.cronjob }} in namespace {{ $labels.namespace }}
            has a success rate of {{ $value | humanizePercentage }}.

      # Alert when no successful execution in 48 hours
      - alert: CronJobNoRecentSuccess
        expr: time() - cronjob_monitor_last_success_timestamp > 172800
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "CronJob {{ $labels.cronjob }} no success in 48h"
          description: >
            CronJob {{ $labels.cronjob }} in namespace {{ $labels.namespace }}
            has not completed successfully in over 48 hours.
```

---

## Helm Chart

### Chart.yaml

```yaml
apiVersion: v2
name: k8s-cronjob-monitor
description: Dead-simple CronJob monitoring for Kubernetes
type: application
version: 0.1.0
appVersion: "0.1.0"
home: https://github.com/kubeshield/k8s-cronjob-monitor
sources:
  - https://github.com/kubeshield/k8s-cronjob-monitor
maintainers:
  - name: KubeShield
    url: https://kubeshield.io
keywords:
  - kubernetes
  - cronjob
  - monitoring
  - prometheus
  - grafana
```

### values.yaml

```yaml
image:
  repository: ghcr.io/kubeshield/k8s-cronjob-monitor
  tag: latest
  pullPolicy: IfNotPresent

replicaCount: 1

# Namespace filtering
namespaces:
  include: []       # Empty = all namespaces
  exclude:
    - kube-system

# Missed schedule detection
scheduling:
  gracePeriodMinutes: 5
  checkIntervalSeconds: 60

# Logging
logging:
  level: info       # debug, info, warn, error
  format: json      # json, console

# Prometheus ServiceMonitor (for Prometheus Operator users)
serviceMonitor:
  enabled: true
  interval: 30s
  namespace: ""     # If empty, uses release namespace

# Resource limits
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi

# Pod configuration
serviceAccount:
  create: true
  name: k8s-cronjob-monitor

nodeSelector: {}
tolerations: []
affinity: {}
```

### NOTES.txt (shown after helm install)

```
🎉 k8s-cronjob-monitor is now running!

Your CronJobs are being auto-discovered and monitored.

📊 Next steps:

1. Import Grafana dashboards:
   Dashboard ID: XXXXX (will be assigned on Grafana.com)
   Or import from: https://github.com/kubeshield/k8s-cronjob-monitor/tree/main/dashboards

2. Verify metrics:
   kubectl port-forward svc/{{ .Release.Name }} 8080:8080 -n {{ .Release.Namespace }}
   curl http://localhost:8080/metrics | grep cronjob_monitor

3. Configure alerts (optional):
   kubectl apply -f https://raw.githubusercontent.com/kubeshield/k8s-cronjob-monitor/main/alerts/cronjob-alerts.yaml

📖 Documentation: https://github.com/kubeshield/k8s-cronjob-monitor

🔒 Need compliance automation for SOC2/HIPAA?
   Check out KubeShield → https://kubeshield.io
```

---

## RBAC & Security

### Minimal Read-Only Permissions

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k8s-cronjob-monitor
rules:
  # Read CronJobs (for auto-discovery)
  - apiGroups: ["batch"]
    resources: ["cronjobs"]
    verbs: ["get", "list", "watch"]

  # Read Jobs (to track executions)
  - apiGroups: ["batch"]
    resources: ["jobs"]
    verbs: ["get", "list", "watch"]

  # Read Pods (for exit codes and status)
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]

  # Read Pod logs (optional, for failure details)
  - apiGroups: [""]
    resources: ["pods/log"]
    verbs: ["get"]

# What the operator does NOT need:
# ❌ No write permissions
# ❌ No delete permissions
# ❌ No secrets access
# ❌ No cluster-admin
# ❌ No kube-system access
```

### Security Posture
- Read-only access to the Kubernetes API
- No network egress required (data stays in cluster)
- No persistent storage needed
- Runs as non-root user
- No privilege escalation
- Distroless base image (minimal attack surface)

---

## Installation Guide

### Prerequisites
- Kubernetes cluster (1.19+)
- Helm 3.0+
- Prometheus (for metrics scraping)
- Grafana (for dashboards)

### Quick Start

```bash
# Add Helm repository
helm repo add k8s-cronjob-monitor https://kubeshield.github.io/k8s-cronjob-monitor
helm repo update

# Install
helm install k8s-cronjob-monitor k8s-cronjob-monitor/k8s-cronjob-monitor \
  --namespace monitoring \
  --create-namespace

# That's it! CronJobs are now being monitored.
```

### Verify Installation

```bash
# Check operator is running
kubectl get pods -n monitoring -l app=k8s-cronjob-monitor

# Check metrics endpoint
kubectl port-forward -n monitoring svc/k8s-cronjob-monitor 8080:8080
curl http://localhost:8080/metrics | grep cronjob_monitor

# Verify Prometheus is scraping (Prometheus UI → Status → Targets)
```

### Import Grafana Dashboards

**Method 1: Grafana.com**
1. Open Grafana → "+" → "Import"
2. Enter dashboard ID: `XXXXX` (will be assigned)
3. Select Prometheus data source → Import

**Method 2: JSON file**
```bash
curl -o cronjob-overview.json \
  https://raw.githubusercontent.com/kubeshield/k8s-cronjob-monitor/main/dashboards/cronjob-overview.json
# Import via Grafana UI or API
```

### Configure Prometheus Scraping

**If using Prometheus Operator (ServiceMonitor auto-created by Helm):**
No action needed — the ServiceMonitor is created automatically.

**If using vanilla Prometheus:**
```yaml
# Add to prometheus.yaml scrape_configs
scrape_configs:
  - job_name: 'k8s-cronjob-monitor'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - monitoring
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: k8s-cronjob-monitor
      - source_labels: [__meta_kubernetes_pod_ip]
        action: replace
        target_label: __address__
        replacement: $1:8080
```

---

## Development Setup

### Local Development with kind

```bash
# 1. Clone repository
git clone https://github.com/kubeshield/k8s-cronjob-monitor.git
cd k8s-cronjob-monitor

# 2. Install dependencies
go mod download

# 3. Create local cluster with kind (includes Prometheus + Grafana + sample CronJobs)
make kind-create

# 4. Run operator locally (against the kind cluster)
make run

# 5. In another terminal, port-forward Grafana
kubectl port-forward -n monitoring svc/grafana 3000:80
# Open http://localhost:3000 (admin/admin)

# 6. Import dashboards
make import-dashboards

# 7. Create test CronJobs
kubectl apply -f examples/basic-cronjob.yaml

# 8. Trigger a manual job to see metrics immediately
kubectl create job --from=cronjob/test-success test-manual-1

# 9. Run tests
make test

# 10. Build container image
make docker-build
```

### Makefile Targets

```makefile
make run              # Run operator locally
make test             # Run unit tests
make test-e2e         # Run end-to-end tests
make lint             # Run golangci-lint
make docker-build     # Build container image
make docker-push      # Push to GHCR
make kind-create      # Create kind cluster with monitoring stack
make kind-delete      # Delete kind cluster
make import-dashboards # Import Grafana dashboards
make helm-package     # Package Helm chart
make release          # Full release (build, tag, push, chart)
```

---

## Testing Strategy

### Unit Tests (Go standard testing)
- Metric collection logic
- Schedule parsing and missed schedule detection
- Owner reference resolution
- Namespace filtering

### Integration Tests
- Informer event handling with fake Kubernetes client
- Prometheus metric endpoint validation
- ServiceMonitor configuration

### End-to-End Tests (kind cluster)
- Deploy operator in kind cluster
- Create CronJobs and verify metrics appear
- Trigger job failures and verify failure metrics
- Verify Grafana dashboard loads with data
- Test namespace filtering

### CI Pipeline (.github/workflows/ci.yaml)
```yaml
- Run golangci-lint
- Run unit tests with coverage (>80%)
- Build Docker image
- Create kind cluster
- Deploy operator
- Run e2e tests
- Upload coverage report
```

---

## Development Phases

### Phase 1: Operator Core (Week 1)
- Kubebuilder scaffolding (no CRDs needed — watches built-in resources)
- CronJob watcher with Informers
- Job watcher for execution tracking
- Prometheus metrics exporter (all metrics defined above)
- Basic logging with zap
- Unit tests (>80% coverage)
- Dockerfile (distroless base)
- Makefile

**Success criteria**: Operator runs, discovers CronJobs, exports correct metrics

### Phase 2: Dashboards, Alerts & Helm (Week 2)
- Missed schedule detection logic
- Grafana dashboard JSON (overview + details)
- Prometheus alert rules YAML
- Helm chart with ServiceMonitor
- Namespace filtering
- kind-based local development setup
- Integration tests

**Success criteria**: Full monitoring pipeline works end-to-end in kind cluster

### Phase 3: Polish & Launch (Week 3)
- README with feature comparison table, demo GIF
- Documentation (installation, configuration, metrics reference, troubleshooting, architecture)
- CONTRIBUTING.md
- GitHub Actions CI/CD (build, test, release, Helm publish)
- End-to-end tests
- Container image published to GHCR
- Helm chart published to GitHub Pages
- Cross-promotion for KubeShield Compliance Platform

**Success criteria**: Anyone can install in <60 seconds and see their CronJobs monitored

---

## Launch Plan

### Pre-Launch (During Week 3)
- Create GitHub repository (kubeshield/k8s-cronjob-monitor)
- Set up GitHub Pages for Helm chart hosting
- Write launch blog post: "Why We Built a Free CronJob Monitor"
- Record demo GIF/video for README
- Prepare social media posts

### Launch Day
1. **GitHub**: Publish repository, pin to organization profile
2. **Hacker News**: "Show HN: Free, open-source Kubernetes CronJob monitoring"
3. **Reddit**: Post to r/kubernetes, r/devops, r/selfhosted
4. **Product Hunt**: Launch with demo video
5. **Twitter/X**: Thread explaining the problem and solution

### Post-Launch (Week 4+)
- Submit to CNCF Landscape (under Observability)
- List on Artifact Hub
- Publish Grafana dashboard to Grafana.com
- Post in Kubernetes Slack channels (#monitoring, #general)
- Write tutorial: "Monitor All Your CronJobs in 60 Seconds"
- Engage with issues and PRs

---

## Success Metrics

### Adoption (12-month targets)
- GitHub stars: 1,000+
- Docker Hub / GHCR pulls: 5,000+
- Helm installs: 2,000+
- Active installations (opt-in telemetry): 500+
- Community contributors: 15+

### Quality
- Installation success rate: >95%
- Memory usage: <50MB
- CPU usage: <0.05 cores
- Metric accuracy: >99%
- Zero false positives for missed schedule detection

### Strategic (Lead Generation for KubeShield)
- Email signups via docs/README: 100+
- Click-throughs to KubeShield: 500+
- KubeShield trials from CronJob monitor users: 20+

---

## Competitive Positioning

| Feature | k8s-cronjob-monitor | Cronitor | Healthchecks.io | DIY Prometheus |
|---------|---------------------|----------|-----------------|----------------|
| **Cost** | **Free** | $21–449/mo | $0–80/mo | Free (20hr setup) |
| **Setup time** | **60 seconds** | 10+ min per job | 10+ min per job | Hours |
| **Auto-discovery** | **Yes** | No | No | No |
| **Data location** | **Your cluster** | Their SaaS | Their SaaS | Your cluster |
| **Kubernetes-native** | **Yes (Informers)** | Partial (agent) | No | Manual |
| **Configuration** | **Zero** | Per-job | Per-job | Extensive |
| **Dashboards** | **Pre-built** | Built-in | Basic | Build yourself |
| **Alert rules** | **Pre-built** | Built-in | Basic | Write yourself |
| **Air-gapped** | **Yes** | No | No | Yes |
| **Open source** | **Apache 2.0** | No | Partially | N/A |

### Why Users Choose Us
1. **Zero cost**: Completely free, no limits
2. **Zero config**: Auto-discovers everything
3. **Zero egress**: Data never leaves the cluster
4. **Fits existing stack**: Works with Prometheus/Grafana they already run
5. **60-second install**: One Helm command and done

---

## Appendix

### Glossary
- **CronJob**: Kubernetes resource for scheduled, recurring jobs
- **Job**: Single execution instance of a CronJob
- **Informer**: Kubernetes client-go mechanism for efficient resource watching
- **controller-runtime**: Go library for building Kubernetes operators
- **ServiceMonitor**: Prometheus Operator CRD for configuring metric scraping
- **kind**: Kubernetes IN Docker — local K8s clusters for testing

### Resources
- [Kubernetes CronJob Documentation](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/)
- [Prometheus Client Go](https://github.com/prometheus/client_golang)
- [Kubebuilder Documentation](https://book.kubebuilder.io/)
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [kind (Kubernetes IN Docker)](https://kind.sigs.k8s.io/)

---

*Document Version: 2.0*
*Last Updated: February 2026*
*Status: Active Development*
*Build Approach: Claude Code assisted development*
