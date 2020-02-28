VERSION ?= v0.1.7
# Image URL to use all building/pushing image targets
IMG_REG ?= registry.cn-hangzhou.aliyuncs.com/r2d2
# IMG_REG ?= registry.cn-shanghai.aliyuncs.com/zhd173
IMG_CTL := $(IMG_REG)/sym-admin-controller
IMG_API := $(IMG_REG)/sym-admin-api
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# This repo's root import path (under GOPATH).
ROOT := gitlab.dmall.com/arch/sym-admin

GO_VERSION := 1.13.6
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

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: manager-controller manager-api

manager-controller: generate fmt
	GOOS=linux GOARCH=amd64 go build -o bin/sym-admin-controller -ldflags "-s -w -X $(ROOT)/pkg/version.Release=$(VERSION) -X $(ROOT)/pkg/version.Commit=$(COMMIT) -X $(ROOT)/pkg/version.BuildDate=$(BUILD_DATE)" cmd/controller/main.go

manager-api: generate fmt
	GOOS=linux GOARCH=amd64 go build -o bin/sym-admin-api -ldflags "-s -w -X $(ROOT)/pkg/version.Release=$(VERSION) -X $(ROOT)/pkg/version.Commit=$(COMMIT) -X $(ROOT)/pkg/version.BuildDate=$(BUILD_DATE)" cmd/sym-api/main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run cmd/controller/main.go

# Install CRDs into a cluster
crd: generate manifests
	kustomize build config/crd > manifests/crd.yaml

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

# Generate code, e.g. XXX.deepcopy.go
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Build the docker image
docker-build:
	docker run --rm -v "$$PWD":/go/src/${ROOT} -w /go/src/${ROOT} golang:${GO_VERSION} make build

build:
	$(GO) -v -o bin/sym-admin-controller -ldflags "-s -w -X $(ROOT)/pkg/version.Release=$(VERSION) -X  $(ROOT)/pkg/version.Commit=$(COMMIT)   \
	-X  $(ROOT)/pkg/version.BuildDate=$(BUILD_DATE)" cmd/controller/main.go
	$(GO) -v -o bin/sym-admin-api -ldflags "-s -w -X  $(ROOT)/pkg/version.Release=$(VERSION) -X  $(ROOT)/pkg/version.Commit=$(COMMIT)   \
	-X  $(ROOT)/pkg/version.BuildDate=$(BUILD_DATE)" cmd/sym-api/main.go

docker-push: docker-push-controller docker-push-api

# Push the docker image
docker-push-controller: manager-controller
	docker build -t ${IMG_CTL}:${VERSION} -f ./install/Dockerfile-ctl .
	docker push ${IMG_CTL}:${VERSION}

docker-push-api: docker-build
	docker build -t ${IMG_API}:${VERSION} -f ./install/Dockerfile-api .
	docker push ${IMG_API}:${VERSION}

docker-push-release: docker-build
	docker build -t ${IMG_CTL}:${VERSION} -f ./install/Dockerfile-ctl .
	docker push ${IMG_CTL}:${VERSION}

helm-master:
	helm upgrade --install sym-ctl --namespace sym-admin --set image.tag=${VERSION},image.worker=false,image.master=true ./install/Kubernetes/helm/controller

helm-master-worker:
	helm upgrade --install sym-ctl --namespace sym-admin --set image.tag=${VERSION},image.worker=true,image.master=true ./install/Kubernetes/helm/controller

helm-worker:
	helm upgrade --install sym-ctl --namespace sym-admin --set image.tag=${VERSION},image.worker=true,image.master=false ./install/Kubernetes/helm/controller

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
