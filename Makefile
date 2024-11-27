.DEFAULT_GOAL:=build

BINARY?=discover-workload
VERSION=$(shell git rev-parse HEAD)
RELEASE_TAG ?= "0.0.0"

PLATFORMS=linux
ARCHITECTURES=amd64 arm64 ppc64le s390x

BIN_DIR=bin

RUN_FLAGS?="--help"
.PHONY: run
run:
	go run ./internal/cmd/main/discover-workload.go $(RUN_FLAGS)

.PHONY: build
build:
	CGO_ENABLED=0 go build -o $(BIN_DIR)/$(BINARY) -ldflags "-X github.com/opdev/discover-workload/internal/version.Commit=$(VERSION) -X github.com/opdev/discover-workload/internal/version.Version=$(RELEASE_TAG)" internal/cmd/main/discover-workload.go
	@ls "$(BIN_DIR)" | grep -e '$(BINARY)' &> /dev/null


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
fmt: gofumpt
	${GOFUMPT} -l -w .

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
	 -cover -coverprofile=coverage.out

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter checks.
	$(GOLANGCI_LINT) run

.PHONY: clean
clean:
	@go clean
	@# cleans the binary created by make build
	$(shell if [ -f "$(BINARY)" ]; then rm -f $(BINARY); fi)
	@# cleans all the binaries created by make build-multi-arch
	$(foreach GOOS, $(PLATFORMS),\
	$(foreach GOARCH, $(ARCHITECTURES),\
	$(shell if [ -f "$(BINARY)-$(GOOS)-$(GOARCH)" ]; then rm -f $(BINARY)-$(GOOS)-$(GOARCH); fi)))

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
GOLANGCI_LINT_VERSION ?= v1.61.0
golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT):
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION))

GOFUMPT = $(shell pwd)/bin/gofumpt
gofumpt: ## Download envtest-setup locally if necessary.
	$(call go-install-tool,$(GOFUMPT),mvdan.cc/gofumpt@latest)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-install-tool
@[ -f $(1) ] || { \
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
}
endef