# k8s-cronjob-monitor Makefile

# Image URL
IMG ?= ghcr.io/kubeshield/k8s-cronjob-monitor:latest

# Get the currently used golang install path
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: test
test: fmt vet ## Run tests
	go test ./... -coverprofile cover.out

.PHONY: test-coverage
test-coverage: test ## Run tests with coverage report
	go tool cover -html=cover.out -o coverage.html
	@echo "Coverage report: coverage.html"

##@ Build

.PHONY: build
build: fmt vet ## Build binary
	go build -o bin/operator cmd/operator/main.go

.PHONY: run
run: fmt vet ## Run operator locally
	go run cmd/operator/main.go

.PHONY: docker-build
docker-build: ## Build docker image
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image
	docker push ${IMG}

##@ Deployment

.PHONY: deploy
deploy: ## Deploy to cluster using Helm
	helm upgrade --install k8s-cronjob-monitor ./deploy/helm/k8s-cronjob-monitor \
		--namespace monitoring \
		--create-namespace

.PHONY: undeploy
undeploy: ## Remove from cluster
	helm uninstall k8s-cronjob-monitor --namespace monitoring

.PHONY: manifests
manifests: ## Generate raw YAML manifests from Helm
	helm template k8s-cronjob-monitor ./deploy/helm/k8s-cronjob-monitor \
		--namespace monitoring > deploy/manifests/operator.yaml

##@ Local Testing

.PHONY: kind-create
kind-create: ## Create kind cluster with monitoring stack
	./hack/test-setup.sh create

.PHONY: kind-delete
kind-delete: ## Delete kind cluster
	kind delete cluster --name cronjob-monitor-test

.PHONY: import-dashboards
import-dashboards: ## Import Grafana dashboards
	./hack/import-dashboards.sh

.PHONY: test-cronjobs
test-cronjobs: ## Deploy test CronJobs
	kubectl apply -f examples/
