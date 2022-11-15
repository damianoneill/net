# Check for required command tools to build or stop immediately
EXECUTABLES ?= git go find xargs pwd curl awk docker wget grep sed
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))
		
MAKEFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
ABS_CURRENT_DIR := $(dir $(MAKEFILE_PATH))
CURRENT_DIR := $(notdir $(patsubst %/,%,$(dir $(MAKEFILE_PATH))))

# Go related variables
GOBIN := $(CURDIR)/.go/bin
export PATH := $(GOBIN):$(PATH)
CGO_ENABLED ?= 0 # disabled, override as env variable
GO := CGO_ENABLED=$(CGO_ENABLED) go
GOPATH ?= $(shell $(GO) env GOPATH)
GOFMT := $(GOBIN)/gofumpt -w
GOMODULE := $(shell $(GO) list)
GOTOOLS := $(shell cat tools.go | grep _ | awk -F'"' '{print $2}' | sed "s/.*\///" | sed -e 's/^"//' -e 's/"$$//' | awk '{print "$(GOBIN)/" $$0}')
 BINARY=$(shell go list -f '{{.Target}}') 

# variables passed to binary
LD_VERSION = x.x.x
LD_COMMIT = 001
ifeq ($(shell git rev-parse --is-inside-work-tree 2>/dev/null),true)
LD_VERSION = $(shell git describe --tags --abbrev=0 --dirty=-next 2>/dev/null)
LD_COMMIT = $(shell git rev-parse HEAD)
endif
LD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LD_FLAGS := -s -w -X $(GOMODULE)/cmd.version=$(LD_VERSION) -X $(GOMODULE)/cmd.commit=$(LD_COMMIT) -X $(GOMODULE)/cmd.date=$(LD_DATE)

# third party versions
GOLANGCI_LINT_VERSION := v1.49.0
TRIVY_VERSION=$(shell wget -qO - "https://api.github.com/repos/aquasecurity/trivy/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')


.DEFAULT_GOAL := all

.PHONY: all-default
all-default: ## run the tools mod, generate, fmt, test, lint and install targets
all-default: tools mod generate fmt test lint install

.PHONY: install-default
install-default: ## install the binary
	@echo ">>> go install "
	@$(GO) install -ldflags="$(LD_FLAGS)" ./...

.PHONY: build-default
build-default: ## build the binary, ignoring vendored code if it exists
	@echo ">>> go build "
	@$(GO) build -ldflags="$(LD_FLAGS)" ./...

.PHONY: test-default
test-default: ## run test with coverage
	@echo ">>> go test "
	@$(GO) test -v -cover ./... -coverprofile cover.out

.PHONY: coverage-default
coverage-default: ## report on test coverage
coverage-default: test
	@echo ">>> govereport "
	@$(GOBIN)/goverreport -coverprofile=cover.out -sort=block -order=desc -threshold=85

.PHONY: fmt-default
fmt-default: ## format the code
	@echo ">>> gofmt"
	@$(GOFMT) .

.PHONY: mod-default
mod-default: ## makes sure go.mod matches the source code in the module
	@echo ">>> go mod tidy "
	@$(GO) mod tidy

.PHONY: archive-default
archive-default: ## archive the third party dependencies, typically prior to generating a tagged release
	@echo ">>> go mod vendor"
	@$(GO) mod vendor

.PHONY: lint-default
lint-default: ## run golangci-lint using the configuration in .golangci.yml
	@echo ">>> golangci-lint run "
	@$(GOBIN)/golangci-lint run

.PHONY: generate-default
generate-default: ## go generate code
	@echo ">>> go generate "
	@$(GO) generate ./...

tools-default: ## install the project specific tools into $GOBIN
tools-default: mod $(GOBIN)/golangci-lint $(GOBIN)/trivy $(GOTOOLS)

$(GOTOOLS):
	@echo ">>> install from tools.go "
	@cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % sh -c 'GOBIN=$(GOBIN) $(GO) install %'

$(GOBIN)/golangci-lint:
	@echo ">>> install golangci-lint "
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) $(GOLANGCI_LINT_VERSION)

$(GOBIN)/trivy:
	@echo ">>> install trivy, be patient "
	@mkdir -p $(GOPATH)/src/github.com/aquasecurity
	@cd $(GOPATH)/src/github.com/aquasecurity ; \
	rm -rf trivy ; \
	git clone --depth 1 --branch v$(TRIVY_VERSION) https://github.com/aquasecurity/trivy 2>/dev/null; \
	cd trivy/cmd/trivy/ ; \
	export GO111MODULE=on ; \
	GOBIN=$(GOBIN) $(GO) install ; \
	cd $(ABS_CURRENT_DIR)

.PHONY: runner-default
runner-default: ## execute the gitlab runner using the configuration in .gitlab-ci.yml
	gitlab-runner exec docker --cache-dir /cache --docker-volumes 'cache:/cache' test

.PHONY: licenses-default
licenses-default: ## print list of licenses for third party software used in binary
licenses-default: install
	@echo ">>> lichen "
	@$(GOBIN)/lichen --config=lichen.yaml $(GOPATH)/bin/$(CURRENT_DIR)

.PHONY: add-license-default
add-license-default: ## add copyright license headers to go source code
add-license-default:
	@echo ">>> addlicense "
	find . -type f -name "*.go" | xargs $(GOBIN)/addlicense -c "acme ltd" -l "apache" -y "2021"

.PHONY: security-default
security-default: ## run go security check
security-default:
	@echo ">>> gosec "
	@$(GOBIN)/gosec -conf .gosec.json ./...

.PHONY: scanner-default
scanner-default: ## run go vulnerability scanner
scanner-default: install-default
	@echo ">>> trivy "
	$(GOBIN)/trivy fs --vuln-type library $(BINARY)

.PHONY: outdated-default
outdated-default: ## check for outdated direct dependencies
outdated-default:
	@echo ">>> go-mod-outdated "
	@$(GO) list -u -m -json all | go-mod-outdated -direct

.PHONY: lines-default
lines-default: ## shorten lines longer than 100 chars, ignore generated
lines-default:
	@echo ">>> golines "
	@$(GOBIN)/golines --ignore-generated -m 100 -w .

.PHONY: authors-default
authors-default: ## update the AUTHORS file
authors-default:
	@echo ">>> authors "
	@git log --all --format='%aN <%aE>' | sort -u | egrep -v noreply > AUTHORS

.PHONY: changelog-default
changelog-default: ## update the CHANGELOG.md
changelog-default:
	@echo ">>> changelog "
	@$(GOBIN)/git-chglog -o CHANGELOG.md

.PHONY: help-default
help-default:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m ignore suffix -default e.g. make install\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

%:  %-default
	@  true
