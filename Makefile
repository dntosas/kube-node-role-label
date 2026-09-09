PROJECT     := kube-node-role-label
MODULE      := github.com/dntosas/$(PROJECT)
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
VERSION     := $(shell git describe --tags --exact-match --match 'v*' 2>/dev/null || echo development)
IMAGE       ?= ghcr.io/dntosas/$(PROJECT)
CHART_DIR   := charts/$(PROJECT)

LDFLAGS     := -s -w -X $(MODULE)/cmd.Version=$(VERSION) -X $(MODULE)/cmd.CommitHash=$(COMMIT)
GOFLAGS     := -trimpath

GOLANGCI_LINT_VERSION ?= v2.13.2
HELM_DOCS_VERSION     ?= v1.14.2

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Format Go code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet.
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (installs the pinned version if missing).
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

.PHONY: test
test: ## Run unit tests with the race detector.
	go test -race -count=1 -coverprofile=cover.out ./...

.PHONY: vulncheck
vulncheck: ## Scan dependencies for known vulnerabilities.
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: tidy
tidy: ## go mod tidy and verify.
	go mod tidy
	go mod verify

.PHONY: ci
ci: fmt vet lint test ## Everything CI runs.

##@ Build

.PHONY: build
build: ## Build the binary for the host platform into bin/.
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o bin/$(PROJECT) .

.PHONY: run
run: ## Run against the current kubeconfig context. Usage: make run LABELS=node-type
	go run . -label $(LABELS) -log-format text -v

.PHONY: docker-build
docker-build: ## Build the container image for the host architecture.
	scripts/docker-build.sh $(IMAGE):$(VERSION)

.PHONY: release-snapshot
release-snapshot: ## Dry-run the goreleaser pipeline locally (binaries + archives; images are built at publish time).
	go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean --skip=publish --config .github/config/goreleaser.yaml

.PHONY: release-check
release-check: ## Validate the goreleaser configuration.
	go run github.com/goreleaser/goreleaser/v2@latest check --config .github/config/goreleaser.yaml

##@ Helm

.PHONY: helm-lint
helm-lint: ## Lint and render the chart.
	helm lint --strict $(CHART_DIR)
	helm template $(PROJECT) $(CHART_DIR) > /dev/null

.PHONY: helm-docs
helm-docs: ## Regenerate the chart README from values.yaml.
	go run github.com/norwoodj/helm-docs/cmd/helm-docs@$(HELM_DOCS_VERSION) --chart-search-root=charts

.PHONY: clean
clean: ## Remove build artifacts.
	rm -rf bin dist cover.out $(PROJECT)
