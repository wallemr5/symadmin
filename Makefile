VERSION ?= v1.1.0-dev10
# Image URL to use all building/pushing image targets
IMG_REG ?= symcn.tencentcloudcr.com/symcn
IMG_CTL := $(IMG_REG)/sym-admin-controller
IMG_API := $(IMG_REG)/sym-admin-api
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

KUBECONFIG ?= "./manifests/kubeconfig.yaml"

# This repo's root import path (under GOPATH).
ROOT := gitlab.dmall.com/arch/sym-admin

GO_VERSION := 1.14.6
ARCH     ?= $(shell go env GOARCH)
BUILD_DATE = $(shell date +'%Y-%m-%dT%H:%M:%SZ')
COMMIT    = $(shell git rev-parse --short HEAD)
GOENV    := CGO_ENABLED=0 GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(ARCH) GOPROXY=https://goproxy.io,direct
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
test: set-goproxy fmt vet
	go test -race -cover ./...


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

# Speed up Go module downloads in CI
set-goproxy:
	go env -w GOPROXY=https://goproxy.io,direct

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
	docker run --rm -v "$$PWD":/go/src/${ROOT} -v ${GOPATH}/pkg/mod:/go/pkg/mod -w /go/src/${ROOT} golang:${GO_VERSION} make build

docker-build-controller:
	docker run --rm -v "$$PWD":/go/src/${ROOT} -v ${GOPATH}/pkg/mod:/go/pkg/mod -w /go/src/${ROOT} golang:${GO_VERSION} make build-controller

docker-build-api:
	docker run --rm -v "$$PWD":/go/src/${ROOT} -v ${GOPATH}/pkg/mod:/go/pkg/mod -w /go/src/${ROOT} golang:${GO_VERSION} make build-api

build: build-controller build-api

build-controller:
	$(GO) -v -o bin/sym-admin-controller -ldflags "-s -w -X $(ROOT)/pkg/version.Release=$(VERSION) -X  $(ROOT)/pkg/version.Commit=$(COMMIT)   \
	-X  $(ROOT)/pkg/version.BuildDate=$(BUILD_DATE)" cmd/controller/main.go

build-api:
	$(GO) -v -o bin/sym-admin-api -ldflags "-s -w -X  $(ROOT)/pkg/version.Release=$(VERSION) -X  $(ROOT)/pkg/version.Commit=$(COMMIT)   \
	-X  $(ROOT)/pkg/version.BuildDate=$(BUILD_DATE)" cmd/sym-api/main.go

docker-push: docker-push-controller docker-push-api

# Push the docker image
docker-push-controller:
	docker build -t ${IMG_CTL}:${VERSION} -f ./docker/Dockerfile-ctl .
	docker push ${IMG_CTL}:${VERSION}

docker-push-api:
	docker build -t ${IMG_API}:${VERSION} -f ./docker/Dockerfile-api .
	docker push ${IMG_API}:${VERSION}

# Install operator with helm
helm-master:
	helm upgrade --install sym-ctl --namespace sym-admin --set image.tag=${VERSION},image.worker=false,image.master=true ./chart/sym-controller

helm-master-worker:
	helm upgrade --install sym-ctl --namespace sym-admin --set image.tag=${VERSION},image.worker=true,image.master=true ./chart/sym-controller

helm-worker:
	helm upgrade --install sym-ctl --namespace sym-admin --set image.tag=${VERSION},image.worker=true,image.master=false ./chart/sym-controller

helm-api:
	helm upgrade --install sym-api --namespace sym-admin --set image.tag=${VERSION} ./chart/sym-api

helm-cluster:
	helm upgrade --install sym-ctl-cluster --namespace sym-admin --set image.tag=${VERSION},image.cluster=true,image.worker=false,image.master=false,image.leader=false,image.threadiness=1,rbac.name=sym-controller-cluster ./chart/sym-controller

helm-test:
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context test-tke-gz-bj5-bus-01 --install sym-ctl --namespace sym-admin --set image.tag=${VERSION},image.worker=true,image.repos="https://kubernetes-charts.storage.googleapis.com",image.master=true,image.reCreate=true,image.threadiness=1,resources.limits.cpu=1,resources.requests.cpu="500m" ./chart/sym-controller
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context dev-tke-gz-bj5-glb --install sym-ctl --namespace sym-admin  --set image.tag=${VERSION},image.repos="https://kubernetes-charts.storage.googleapis.com",image.worker=true,image.master=true,image.reCreate=true,image.threadiness=1,resources.limits.cpu=1,resources.requests.cpu="500m" ./chart/sym-controller
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context cn-tke-bj5-test-01 --install sym-ctl --namespace sym-admin  --set image.tag=${VERSION},image.worker=true,image.master=true,image.reCreate=true,image.repos="https://kubernetes-charts.storage.googleapis.com",image.threadiness=1,resources.limits.cpu=1,resources.requests.cpu="500m" ./chart/sym-controller
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context dev-tke-bj5-monit-01 --install sym-ctl --namespace sym-admin  --set image.tag=${VERSION},image.worker=true,image.master=true,image.reCreate=true,image.repos="https://kubernetes-charts.storage.googleapis.com",image.threadiness=1,resources.limits.cpu=1,resources.requests.cpu="500m" ./chart/sym-controller
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context test-tke-rz-bj5-bus-01 --install sym-ctl --namespace sym-admin  --set image.tag=${VERSION},image.worker=true,image.master=false,image.reCreate=true,image.repos="https://kubernetes-charts.storage.googleapis.com",image.threadiness=1,resources.limits.cpu=1,resources.requests.cpu="500m" ./chart/sym-controller
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context test-tke-rz-cd-bus-01 --install sym-ctl --namespace sym-admin  --set image.tag=${VERSION},image.worker=true,image.repos="https://kubernetes-charts.storage.googleapis.com",image.master=false,image.reCreate=true,image.threadiness=1,resources.limits.cpu=1,resources.requests.cpu="500m" ./chart/sym-controller
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context cn-tke-cd-test-01 --install sym-ctl --namespace sym-admin  --set image.tag=${VERSION},image.worker=true,image.repos="https://kubernetes-charts.storage.googleapis.com",image.master=false,image.reCreate=true,image.threadiness=1,resources.limits.cpu=1,resources.requests.cpu="500m" ./chart/sym-controller
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context dev-tke-bj5-test-01 --install sym-ctl --namespace sym-admin  --set image.tag=${VERSION},image.repos="https://kubernetes-charts.storage.googleapis.com",image.worker=true,image.master=false,image.reCreate=true,image.threadiness=1,resources.limits.cpu=1,resources.requests.cpu="500m" ./chart/sym-controller
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context dev-tke-rz-cd-glb --install sym-ctl --namespace sym-admin  --set image.tag=${VERSION},image.worker=true,image.repos="https://kubernetes-charts.storage.googleapis.com",image.master=false,image.reCreate=true,image.threadiness=1,resources.limits.cpu=1,resources.requests.cpu="500m" ./chart/sym-controller
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context cls-0doi9yrf-context-default --install sym-ctl --namespace sym-admin  --set image.tag=${VERSION},image.worker=true,image.repos="https://kubernetes-charts.storage.googleapis.com",image.master=false,image.reCreate=true,image.threadiness=1,resources.limits.cpu=1,resources.requests.cpu="500m" ./chart/sym-controller
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context test-tke-gz-bj5-bus-01 --install sym-api --namespace sym-admin --set image.tag=${VERSION},ingress.hosts[0].host=testapi.sym.dmall.com,ingress.hosts[0].paths[0]=/,resources.limits.cpu=1,resources.requests.cpu="500m",replicaCount=1 ./chart/sym-api
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context cn-tke-bj5-test-01 --install sym-api --namespace sym-admin --set image.tag=${VERSION},ingress.hosts[0].host=testapi-djj.sym.dmall.com,ingress.hosts[0].paths[0]=/,resources.limits.cpu=1,resources.requests.cpu="500m",replicaCount=1  ./chart/sym-api
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context dev-tke-bj5-monit-01 --install sym-api --namespace sym-admin --set image.tag=${VERSION},ingress.hosts[0].host=devapi.sym.dmall.com,ingress.hosts[0].paths[0]=/,resources.limits.cpu=1,resources.requests.cpu="500m",replicaCount=1 ./chart/sym-api
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context dev-tke-gz-bj5-glb --install sym-api --namespace sym-admin --set image.tag=${VERSION},ingress.hosts[0].host=testapi-glb.sym.dmall.com,ingress.hosts[0].paths[0]=/,resources.limits.cpu=1,resources.requests.cpu="500m",replicaCount=1 ./chart/sym-api
	helm upgrade --kubeconfig ${KUBECONFIG} --kube-context cls-0doi9yrf-context-default --install sym-api --namespace sym-admin --set image.tag=${VERSION},ingress.hosts[0].host=testapi-glb.sym.dmall.com,ingress.hosts[0].paths[0]=/,resources.limits.cpu=1,resources.requests.cpu="500m",replicaCount=1 ./chart/sym-api

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
