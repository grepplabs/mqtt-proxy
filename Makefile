.DEFAULT_GOAL := build

.PHONY: clean build fmt test

TAG           ?= v0.3.0

BUILD_FLAGS   ?=
BINARY        ?= mqtt-proxy
BRANCH        = $(shell git rev-parse --abbrev-ref HEAD)
REVISION      = $(shell git describe --tags --always --dirty)
BUILD_DATE    = $(shell date +'%Y.%m.%d-%H:%M:%S')
LDFLAGS       ?= -w -s \
				 -X github.com/prometheus/common/version.Version=$(TAG) \
				 -X github.com/prometheus/common/version.Revision=$(REVISION) \
				 -X github.com/prometheus/common/version.Branch=$(BRANCH) \
				 -X github.com/prometheus/common/version.BuildUser=$$USER \
				 -X github.com/prometheus/common/version.BuildDate=${BUILD_DATE}

GOOS          ?= $(if $(TARGETOS),$(TARGETOS),linux)
GOARCH        ?= $(if $(TARGETARCH),$(TARGETARCH),amd64)
BUILDPLATFORM ?= $(GOOS)/$(GOARCH)

LOCAL_IMAGE   ?= local/$(BINARY)
CLOUD_IMAGE   ?= grepplabs/mqtt-proxy:$(TAG)

HELM_BIN	  ?= helm3
HELM_VALUES	  ?= noop
SVC_NAME      ?= mqtt-proxy
SVC_NAMESPACE ?= mqtt
CHART_VERSION = $(shell $(HELM_BIN) show chart charts/mqtt-proxy | egrep '^version' | sed 's/version://' | tr -d '[:space:]')
CHART_PKG     ?= .cr-release-packages

ROOT_DIR      := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

default: build

.PHONY: help
help:
	@grep -E '^[a-zA-Z%_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

vet: ## Go vet
	go vet ./...

check: vet
	gosec ./... 2>&1

lint: ## Lint
	golint $$(go list ./...) 2>&1

test: ## Test
	GO111MODULE=on go test -count=1 -mod=vendor $(BUILD_FLAGS) -v ./...

test.race: ## Test with race detection
	GO111MODULE=on go test -race -count=1 -mod=vendor $(BUILD_FLAGS) -v ./...

build: vet ## Build executable
	CGO_ENABLED=1 GO111MODULE=on go build -mod=vendor -o $(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

.PHONY: os-build
os-build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 GO111MODULE=on go build -mod=vendor -o $(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

.PHONY: docker-build
docker-build:
	docker buildx build --build-arg BUILDPLATFORM=$(BUILDPLATFORM) --build-arg TARGETARCH=$(GOARCH) -f Dockerfile -t $(LOCAL_IMAGE) .

.PHONY: docker-push
docker-push:
	docker buildx build --build-arg BUILDPLATFORM=$(BUILDPLATFORM) --build-arg TARGETARCH=$(GOARCH) -t $(LOCAL_IMAGE) .
	docker tag $(LOCAL_IMAGE) $(CLOUD_IMAGE)
	docker push $(CLOUD_IMAGE)

fmt: ## Go format
	go fmt ./...

clean: ## Clean
	@rm -rf $(BINARY)
	@rm -rf $(CHART_PKG)

.PHONY: deps
deps:
	GO111MODULE=on go get ./...

.PHONY: vendor
vendor: ## Go vendor
	GO111MODULE=on go mod vendor

.PHONY: tidy
tidy: ## Go tidy
	GO111MODULE=on go mod tidy

.PHONY: tag
tag: ## Git tag
	git tag $(TAG)

.PHONY: release-setup
release-setup:
	curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh

.PHONY: release-publish
release-publish: helm-package release-setup
	@[ "${GITHUB_TOKEN}" ] && echo "releasing $(TAG)" || ( echo "GITHUB_TOKEN is not set"; exit 1 )
	git push origin $(TAG)
	REVISION=$(REVISION) BRANCH=$(BRANCH) BUILD_DATE=$(BUILD_DATE) CHART_VERSION=$(CHART_VERSION) $(ROOT_DIR)/bin/goreleaser release --rm-dist

.PHONY: release-snapshot
release-snapshot: helm-package
	REVISION=$(REVISION) BRANCH=$(BRANCH) BUILD_DATE=$(BUILD_DATE) CHART_VERSION=$(CHART_VERSION) $(ROOT_DIR)/bin/goreleaser --debug --rm-dist --snapshot --skip-publish


.PHONY: helm-package
helm-package: ## Package helm chart
	$(HELM_BIN) package $(ROOT_DIR)/charts/mqtt-proxy --app-version $(TAG) --version $(CHART_VERSION) --destination $(ROOT_DIR)/$(CHART_PKG)
	mv $(ROOT_DIR)/$(CHART_PKG)/mqtt-proxy-$(CHART_VERSION).tgz $(ROOT_DIR)/$(CHART_PKG)/mqtt-proxy-$(CHART_VERSION)-chart.tgz
	$(HELM_BIN) repo index $(ROOT_DIR)/$(CHART_PKG) --url https://grepplabs.github.io/mqtt-proxy
	# TODO: --merge  old_charts_dir/index.yaml

.PHONY: helm-template
helm-template: ## Template helm chart
	$(HELM_BIN) template $(SVC_NAME) $(ROOT_DIR)/charts/mqtt-proxy \
	   -f $(ROOT_DIR)/charts/mqtt-proxy/values-$(HELM_VALUES).yaml \
	   --namespace=$(SVC_NAMESPACE)

.PHONY: helm-install
helm-install:  ## Install helm chart
	$(HELM_BIN) upgrade $(SVC_NAME) $(ROOT_DIR)/charts/mqtt-proxy \
	   -f $(ROOT_DIR)/charts/mqtt-proxy/values-$(HELM_VALUES).yaml \
	   --namespace=$(SVC_NAMESPACE) \
	   --install \
	   --create-namespace

.PHONY: helm-test
helm-test: ## Test helm chart
	$(HELM_BIN) test $(SVC_NAME) \
		--namespace=$(SVC_NAMESPACE)


.PHONY: install-tools
install-tools: ## Install tools
	go install github.com/securego/gosec/v2/cmd/gosec@latest
