.PHONY: gifs docker-lint golangci-lint-install glazed-lint-build glazed-lint lint lintmax gosec govulncheck test build docsctl-install docsctl-export docsctl-validate docsctl-check logcopter-generate logcopter-check goreleaser tag-major tag-minor tag-patch release bump-go-go-golems install

all: gifs

VERSION=v0.1.14
GORELEASER_ARGS ?= --skip=sign --snapshot --clean
GORELEASER_TARGET ?= --single-target
GOLANGCI_LINT_VERSION ?= $(shell cat .golangci-lint-version)
GOLANGCI_LINT_BIN ?= $(CURDIR)/.bin/golangci-lint
GLAZED_LINT_BIN ?= /tmp/glazed-lint
GLAZED_LINT_PKG ?= github.com/go-go-golems/glazed/cmd/tools/glazed-lint
GLAZED_LINT_TOOL_VERSION ?= v1.3.5
GLAZED_LINT_FLAGS ?=
GLAZED_LINT_DIRS ?= ./cmd/... ./pkg/...
DOCSCTL_BIN ?= $(CURDIR)/.bin/docsctl
DOCSCTL_PKG ?= github.com/go-go-golems/glazed/cmd/docsctl
DOCSCTL_TOOL_VERSION ?= latest
DOCSCTL_SQLITE_PATH ?= .docsctl/help.sqlite
DOCSCTL_PACKAGE ?= go-go-objects
DOCSCTL_PACKAGE_VERSION ?= dev

TAPES=$(wildcard doc/vhs/*tape)
gifs: $(TAPES)
	for i in $(TAPES); do vhs < $$i; done

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run -v

golangci-lint-install:
	mkdir -p $(dir $(GOLANGCI_LINT_BIN))
	GOBIN=$(dir $(GOLANGCI_LINT_BIN)) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

glazed-lint-build:
	@echo "Building glazed-lint from pinned tool module..."
	@echo "Installing $(GLAZED_LINT_PKG)@$(GLAZED_LINT_TOOL_VERSION)"
	@GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_LINT_TOOL_VERSION)

glazed-lint: glazed-lint-build
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)

lint: golangci-lint-install glazed-lint-build
	$(GOLANGCI_LINT_BIN) run -v
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)

lintmax: golangci-lint-install glazed-lint-build
	$(GOLANGCI_LINT_BIN) run -v --max-same-issues=100
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)

gosec:
	GOWORK=off go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude-generated -exclude=G101,G304,G301,G306 -exclude-dir=.history ./...

govulncheck:
	GOWORK=off go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

test:
	GOWORK=off go test ./...

build:
	GOWORK=off go generate ./...
	GOWORK=off go build ./...

docsctl-install:
	mkdir -p $(dir $(DOCSCTL_BIN))
	GOBIN=$(dir $(DOCSCTL_BIN)) go install $(DOCSCTL_PKG)@$(DOCSCTL_TOOL_VERSION)

docsctl-export:
	mkdir -p $(dir $(DOCSCTL_SQLITE_PATH))
	GOWORK=off go run ./cmd/go-go-objects help export --format sqlite --output-path $(DOCSCTL_SQLITE_PATH)

docsctl-validate: docsctl-install
	$(DOCSCTL_BIN) validate --file $(DOCSCTL_SQLITE_PATH) --package $(DOCSCTL_PACKAGE) --version $(DOCSCTL_PACKAGE_VERSION)

docsctl-check: docsctl-export docsctl-validate

logcopter-generate:
	GOWORK=off go generate ./...

logcopter-check:
	GOWORK=off go tool logcopter-gen -area-prefix go-go-golems.go-go-objects -strip-prefix github.com/go-go-golems/go-go-objects -check ./pkg/...

goreleaser:
	GOWORK=off goreleaser release $(GORELEASER_ARGS) $(GORELEASER_TARGET)

tag-major:
	git tag $(shell svu major)

tag-minor:
	git tag $(shell svu minor)

tag-patch:
	git tag $(shell svu patch)

release:
	git push origin --tags
	GOWORK=off GOPROXY=proxy.golang.org go list -m github.com/go-go-golems/go-go-objects@$(shell svu current)

bump-go-go-golems:
	@deps="$$(awk '/^require[[:space:]]+github\.com\/go-go-golems\// { print $$2 } /^[[:space:]]*github\.com\/go-go-golems\// { print $$1 }' go.mod | sort -u)"; \
	if [ -z "$$deps" ]; then \
		echo "No github.com/go-go-golems dependencies in go.mod"; \
	else \
		echo "Bumping go-go-golems dependencies:"; \
		echo "$$deps"; \
		for dep in $$deps; do GOWORK=off go get "$${dep}@latest"; done; \
	fi
	GOWORK=off go mod tidy

GO_GO_OBJECTS_BINARY=$(shell which go-go-objects)
install:
	GOWORK=off go build -o ./dist/go-go-objects ./cmd/go-go-objects && \
		cp ./dist/go-go-objects $(GO_GO_OBJECTS_BINARY)
