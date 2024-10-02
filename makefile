
.PHONY: default
default:
	$(MAKE) -s $(IMAGES)

.PHONY: all
all: default

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-45s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

############################################################################
# Variables
############################################################################

IMAGES ?= network-nri-plugin
VERSION ?= latest

# Contrainer Registry
REGISTRY ?= localhost:5000/network-dra
BASE_IMAGE ?= $(REGISTRY)/base-image:$(VERSION)

# Tools
export PATH := $(shell pwd)/bin:$(PATH)
GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
GINKGO = $(shell pwd)/bin/ginkgo
GOFUMPT = $(shell pwd)/bin/gofumpt
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))

BUILD_DIR ?= build
BUILD_STEPS ?= build tag push
BUILD_CMD ?= build
BUILD_ARGS ?= 
BUILD_REGISTRY ?=

OUTPUT_DIR ?= _output

#############################################################################
# Container: Build, tag, push
#############################################################################

.PHONY: build
build:
	docker $(BUILD_CMD) \
	$(BUILD_ARGS) \
	-t $(BUILD_REGISTRY)$(IMAGE):$(VERSION) \
	--build-arg BUILD_VERSION=$(shell git describe --dirty --tags) \
	--build-arg BASE_IMAGE=$(BASE_IMAGE) \
	-f ./$(BUILD_DIR)/$(IMAGE)/Dockerfile .
.PHONY: tag
tag:
	docker tag $(BUILD_REGISTRY)$(IMAGE):$(VERSION) $(REGISTRY)/$(IMAGE):$(VERSION)
.PHONY: push
push:
	docker push $(REGISTRY)/$(IMAGE):$(VERSION)

#############################################################################
##@ Component (Build, tag, push): use VERSION to set the version. Use BUILD_STEPS to set the build steps (build, tag, push)
#############################################################################

.PHONY: network-nri-plugin
network-nri-plugin: ## Build the network-nri-plugin.
	IMAGE=network-nri-plugin $(MAKE) -s $(BUILD_STEPS)

#############################################################################
##@ Testing & Code check
#############################################################################

.PHONY: lint
lint: golangci-lint ## Run linter against golang code.
	$(GOLANGCI_LINT) run ./...

.PHONY: test
test: output-dir envtest setup-test ## Run the Unit tests (read coverage report: go tool cover -html=_output/cover_unit_test.out -o _output/cover_unit_test.html).
	go test -p 1 -race -cover -short -count=1 -coverprofile $(OUTPUT_DIR)/cover_unit_test.out ./...

.PHONY: check
check: lint test ## Run the linter and the Unit tests.

#############################################################################
##@ Code generation
#############################################################################

.PHONY: generate
generate: gofmt generate-controller generate-client generate-lister generate-informer ## Generate all.

.PHONY: gofmt
gofmt: gofumpt ## Run gofumpt.
	$(GOFUMPT) -w .

.PHONY: generate-helm-chart
generate-helm-chart: output-dir ## Generate network-DRA helm charts.
	helm package ./deployments/network-DRA --version $(shell $(MAKE) -s format-version VERSION=$(VERSION)) --destination ./_output/helm

#############################################################################
# Tools
#############################################################################

.PHONY: output-dir
output-dir:
	@mkdir -p $(OUTPUT_DIR)

# https://github.com/golangci/golangci-lint
.PHONY: golangci-lint
golangci-lint:
	$(call go-get-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2)

# https://github.com/onsi/ginkgo
.PHONY: ginkgo
ginkgo:
	$(call go-get-tool,$(GINKGO),github.com/onsi/ginkgo/v2/ginkgo@v2.13.1)

.PHONY: gofumpt
gofumpt:
	$(call go-get-tool,$(GOFUMPT),mvdan.cc/gofumpt@v0.5.0)

.PHONY: print-e2e-skip-focus
print-e2e-skip-focus:
	@focus="" ; \
	for f in $(call get_list,$(E2E_FOCUS)); do \
		focus="$${focus} --focus $${f}" ; \
	done ; \
	printf "$${focus}" ; \
	skip="" ; \
	for f in $(call get_list,$(E2E_SKIP)); do \
		skip="$${skip} --skip $${f}" ; \
	done ; \
	printf "$${skip}" 

define get_list
$$(echo "$(1)" | sed -r 's/ //g' | sed -r 's/,/ /g' )
endef

# go-get-tool will 'go get' any package $2 and install it to $1.
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

# https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
# https://github.com/semver/semver/pull/724
VERSION_REGEX = ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-((0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)(\.(0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*))*))?(\+([0-9a-zA-Z-]+(\.[0-9a-zA-Z-]+)*))?$
.PHONY: format-version
format-version:
	version="$(VERSION)" ; \
	if ! echo "$${version}" | grep -Eq "$(VERSION_REGEX)" ; then \
		version="v0.0.0-$${version}" ; \
	fi ; \
	printf "$${version}"