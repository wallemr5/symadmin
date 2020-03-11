VERSION ?= v0.0.2
# Image URL to use all building/pushing image targets
IMG_REG ?= registry.cn-hangzhou.aliyuncs.com/dmall

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# This repo's root import path (under GOPATH).
ROOT := gitlab.dmall.com/arch/sym-admin/mesh-demo

GO_VERSION := 1.14
ARCH     ?= $(shell go env GOARCH)
BUILD_DATE = $(shell date +'%Y-%m-%dT%H:%M:%SZ')
COMMIT    = $(shell git rev-parse --short HEAD)
GOENV    := CGO_ENABLED=0 GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(ARCH) GOPROXY=https://goproxy.cn,direct
#GO       := $(GOENV) go build -mod=vendor
GO       := $(GOENV) go build

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Build manager binary
manager:

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...


# Build the docker image
docker-build-consumer:
	docker run --rm -v "$$PWD":/go/src/${ROOT} -w /go/src/${ROOT} golang:${GO_VERSION} make build-consumer

docker-build-provider:
	docker run --rm -v "$$PWD":/go/src/${ROOT} -w /go/src/${ROOT} golang:${GO_VERSION} make build-provider

build-consumer:
	$(GO) -v -o bin/consumer -ldflags "-s -w -X $(ROOT)/pkg/version.Release=$(VERSION) -X  $(ROOT)/pkg/version.Commit=$(COMMIT)   \
	-X  $(ROOT)/pkg/version.BuildDate=$(BUILD_DATE)" cmd/consumer/main.go

build-provider:
	$(GO) -v -o bin/provider -ldflags "-s -w -X  $(ROOT)/pkg/version.Release=$(VERSION) -X  $(ROOT)/pkg/version.Commit=$(COMMIT)   \
	-X  $(ROOT)/pkg/version.BuildDate=$(BUILD_DATE)" cmd/provider/main.go

docker-push-provider:
	docker build -t $(IMG_REG)/provider:${VERSION} -f ./misc/provider/Dockerfile .
	docker push $(IMG_REG)/provider:${VERSION}

docker-push-consumer:
	docker build -t $(IMG_REG)/consumer:${VERSION} -f ./misc/consumer/Dockerfile .
	docker push $(IMG_REG)/consumer:${VERSION}



