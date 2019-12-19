VERSION ?= v0.0.1
# Image URL to use all building/pushing image targets
IMG ?= registry.cn-hangzhou.aliyuncs.com/dmall/sym-admin-controller
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# This repo's root import path (under GOPATH).
ROOT := gitlab.dmall.com/arch/sym-admin

GO_VERSION := 1.13.5
ARCH     ?= $(shell go env GOARCH)
BuildDate = $(shell date +'%Y-%m-%dT%H:%M:%SZ')
Commit    = $(shell git rev-parse --short HEAD)
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

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/sym-admin-controller cmd/operator/main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run cmd/operator/main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd >> manifests/crd-AdvDeployment.yaml

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default >> manifests/all-AdvDeployment.yaml

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Build the docker image
docker-build:
	docker run --rm -v "$$PWD":/go/src/${ROOT} -w /go/src/${ROOT} golang:${GO_VERSION} make build

build:
	$(GO) -v -o bin/sym-admin-controller -ldflags "-s -w -X pkg/version.Release=$(VERSION) -X pkg/version.Commit=$(COMMIT)   \
	-X pkg/version.BuildDate=$(BuildDate)" cmd/operator/main.go

# Push the docker image
docker-push:
	docker build -t ${IMG}:${VERSION} -f ./Dockerfile
	docker push ${IMG}:${VERSION}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.4 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
