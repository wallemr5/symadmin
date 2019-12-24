#!/bin/bash

set -x

GOPATH=$(go env GOPATH)
PACKAGE_NAME=gitlab.dmall.com/arch/sym-admin
REPO_ROOT="$GOPATH/src/$PACKAGE_NAME"
DOCKER_REPO_ROOT="/go/src/$PACKAGE_NAME"
DOCKER_CODEGEN_PKG="/go/src/k8s.io/code-generator"
#apiGroups=(appset/v1 migrate/v1)

pushd ${REPO_ROOT}

## Generate ugorji stuff
#rm "$REPO_ROOT"/pkg/apis/devops/v1/*.deepcopy.go
rm "$REPO_ROOT"/pkg/apis/workload/v1beta1/*.deepcopy.go



# for both CRD and EAS types
docker run --rm -ti -u $(id -u):$(id -g) \
  -v "$REPO_ROOT":"$DOCKER_REPO_ROOT" \
  -w "$DOCKER_REPO_ROOT" \
  registry.cn-hangzhou.aliyuncs.com/dmall/gengo:release-1.14 "$DOCKER_CODEGEN_PKG"/generate-groups.sh all \
  gitlab.dmall.com/arch/sym-admin/pkg/client \
  gitlab.dmall.com/arch/sym-admin/pkg/apis \
  "workload:v1beta1" \
  --go-header-file "$DOCKER_REPO_ROOT/hack/boilerplate.go.txt"

popd