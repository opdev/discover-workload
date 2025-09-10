.DEFAULT_GOAL:=build

COMMIT=$(shell git rev-parse HEAD)
BIN_DIR		= bin
BIN_NAME 	?= discover-workload
BIN_VERSION ?= "0.0.0"


RUN_FLAGS	?= "--help"
.PHONY: run
run:
	go run ./internal/cmd/main/discover-workload.go $(RUN_FLAGS)

.PHONY: bin build
bin build:
	CGO_ENABLED=0 go build -o $(BIN_NAME) \
		-trimpath \
		-ldflags "\
			-s -w \
			-X github.com/opdev/discover-workload/internal/version.Commit=$(COMMIT) \
			-X github.com/opdev/discover-workload/internal/version.Version=$(BIN_VERSION)" \
		internal/cmd/main/discover-workload.go

# Fail if git diff detects a change. Useful for CI.
.PHONY: diff-check
diff-check:
	git diff --exit-code

.PHONY: ci.fmt
ci.fmt: fmt diff-check
	echo "=> ci.fmt done"

.PHONY: ci.tidy
ci.tidy: tidy diff-check
	echo "=> ci.tidy done"

.PHONY: fmt
fmt: install.gofumpt
	$(GOFUMPT) -l -w .

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: test
test:
	go test -v ./...

.PHONY: cover
cover:
	go test -v \
	 -race \
	 -cover -coverprofile=coverage.out \
	 ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint: install.golangci-lint
	$(GOLANGCI_LINT) run

# gofumpt
GOFUMPT = $(BIN_DIR)/gofumpt
GOFUMPT_VERSION ?= v0.9.1
install.gofumpt:
	$(call go-install-tool,$(GOFUMPT),mvdan.cc/gofumpt@$(GOFUMPT_VERSION))

# golangci-lint
GOLANGCI_LINT = $(BIN_DIR)/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.4.0
install.golangci-lint:
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-install-tool
@[ -f $(1) ] || { \
GOBIN=$(PROJECT_DIR)/$(BIN_DIR) go install $(2) ;\
}
endef