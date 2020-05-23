.DEFAULT_GOAL := build

.PHONY: clean build fmt test

TAG           ?= "v0.0.1"

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

GOARCH        ?= amd64
GOOS          ?= linux

LOCAL_IMAGE   ?= local/$(BINARY)
CLOUD_IMAGE   ?= grepplabs/mqtt-proxy:$(TAG)

ROOT_DIR      := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

default: build

check:
	go vet ./...
	golint $$(go list ./...) 2>&1
	gosec ./... 2>&1

test:
	GO111MODULE=on go test -mod=vendor -v ./...

build:
	CGO_ENABLED=1 GO111MODULE=on go build -mod=vendor -o $(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

.PHONY: os-build
os-build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 GO111MODULE=on go build -mod=vendor -o $(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

.PHONY: docker-build
docker-build:
	docker build -f Dockerfile -t $(LOCAL_IMAGE) .

.PHONY: docker-push
docker-push:
	docker build -f Dockerfile -t $(LOCAL_IMAGE) .
	docker tag $(LOCAL_IMAGE) $(CLOUD_IMAGE)
	docker push $(CLOUD_IMAGE)

fmt:
	go fmt ./...

clean:
	@rm -rf $(BINARY)

.PHONY: deps
deps:
	GO111MODULE=on go get ./...

.PHONY: vendor
vendor:
	GO111MODULE=on go mod vendor

.PHONY: tidy
tidy:
	GO111MODULE=on go mod tidy

.PHONY: tag
tag:
	git tag $(TAG)

.PHONY: release-setup
release-setup:
	curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh

.PHONY: release-publish
release-publish: release-setup
	@[ "${GITHUB_TOKEN}" ] && echo "releasing $(TAG)" || ( echo "GITHUB_TOKEN is not set"; exit 1 )
	git push origin $(TAG)
	REVISION=$(REVISION) BRANCH=$(BRANCH) BUILD_DATE=$(BUILD_DATE) $(ROOT_DIR)/bin/goreleaser release --rm-dist

.PHONY: release-snapshot
release-snapshot:
	REVISION=$(REVISION) BRANCH=$(BRANCH) BUILD_DATE=$(BUILD_DATE) $(ROOT_DIR)/bin/goreleaser --debug --rm-dist --snapshot --skip-publish
