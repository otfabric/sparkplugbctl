.DEFAULT_GOAL := help

.PHONY: help build install fmt fmt-check lint lint-ci vet vuln test stress coverage cover coverage-gate check clean e2e build-all

help: ## This help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

export GOWORK := off

GO_MOD_VERSION := $(shell sed -nE 's/^go ([0-9]+\.[0-9]+).*/\1/p' go.mod | head -n1)
GOFMT_CI = $(shell GOTOOLCHAIN=go$(GO_MOD_VERSION).0 go env GOROOT)/bin/gofmt

BIN_DIR := bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)

TEST_PKGS := $(shell go list ./...)
COVER_PKGS := $(shell go list ./... | paste -sd, -)
COVER_MIN ?= 90

GO_FILES = $(shell git ls-files --cached --others --exclude-standard -- '*.go')

build: ## Build the sparkplugbctl binary into ./bin
	@echo "Building sparkplugbctl"
	@mkdir -p $(BIN_DIR)
	@go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/sparkplugbctl .

build-all: ## Cross-compile the release platforms into ./bin
	@for p in linux/amd64 linux/arm64 linux/arm darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do \
		os=$${p%/*}; cpu=$${p#*/}; ext=; [ $$os = windows ] && ext=.exe; \
		echo "Building sparkplugbctl-$$os-$$cpu$$ext"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$cpu GOARM=7 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/sparkplugbctl-$$os-$$cpu$$ext . || exit 1; done

e2e: ## Build the binary and run the e2e suite against it
	@mkdir -p bin
	@go build -ldflags "$(LDFLAGS)" -o bin/sparkplugbctl .
	@SPARKPLUGBCTL_E2E_BIN=$(CURDIR)/bin/sparkplugbctl SPARKPLUGBCTL_E2E_VERSION=$(VERSION) go test -count=1 -tags=e2e ./e2e/...

install: ## Install sparkplugbctl into GOPATH/bin
	@echo "Installing sparkplugbctl"
	@go install -ldflags "$(LDFLAGS)" .

fmt: ## Format Go code with the gofmt of the Go version in go.mod
	@echo "Running gofmt (go$(GO_MOD_VERSION))"
	@$(GOFMT_CI) -w $(GO_FILES)

fmt-check: ## Fail if gofmt would change any file
	@echo "Checking formatting (gofmt go$(GO_MOD_VERSION) and local $$(go env GOVERSION))"
	@ci="$$($(GOFMT_CI) -l $(GO_FILES))"; loc="$$(gofmt -l $(GO_FILES))"; \
	if [ -n "$$ci$$loc" ]; then \
		[ -z "$$ci" ] || { echo "Not formatted for gofmt go$(GO_MOD_VERSION) (CI):"; echo "$$ci"; }; \
		[ -z "$$loc" ] || { echo "Not formatted for the local gofmt:"; echo "$$loc"; }; \
		echo "Run 'make fmt'."; \
		exit 1; \
	fi

lint: ## Run staticcheck
	@echo "Running staticcheck"
	@staticcheck ./...

lint-ci: ## Run golangci-lint (see .golangci.yaml)
	@echo "Running golangci-lint"
	@golangci-lint run ./...

vet: ## Run go vet
	@echo "Running go vet"
	@go vet ./...

vuln: ## Run govulncheck
	@echo "Running govulncheck"
	@govulncheck ./...

test: ## Run tests with the race detector
	@echo "Running tests"
	@go test -count=1 -race $(TEST_PKGS)

stress: ## Run tests shuffled, twice, with the race detector
	@echo "Running the tests shuffled, twice, with the race detector"
	@go test -count=2 -race -shuffle=on -timeout=30m $(TEST_PKGS)

coverage: ## Run tests with coverage (writes coverage.out)
	@echo "Running coverage"
	@go test -count=1 -race -coverprofile=coverage.out -covermode=atomic -coverpkg=$(COVER_PKGS) $(TEST_PKGS)
	@go tool cover -func=coverage.out | tail -n1

coverage-gate: coverage ## Fail if coverage is below COVER_MIN percent
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {sub("%", "", $$3); print $$3}'); \
	if awk -v t="$$total" -v m="$(COVER_MIN)" 'BEGIN { exit !(t + 0 >= m + 0) }'; then \
		echo "Coverage $$total% meets the gate of $(COVER_MIN)%"; \
	else \
		echo "Coverage $$total% is below the gate of $(COVER_MIN)%"; exit 1; \
	fi

cover: coverage ## Open coverage report in browser
	@echo "Opening coverage report"
	@go tool cover -html=coverage.out

check: fmt fmt-check lint lint-ci vet vuln test coverage-gate ## Run format + lint + vet + vuln + test + coverage gate

clean: ## Remove generated binaries and coverage artifacts
	@echo "Cleaning up"
	@rm -rf $(BIN_DIR) coverage.out coverage.html
