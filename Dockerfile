FROM registry.cn-hangzhou.aliyuncs.com/dmall/alpine-base:v3.10
MAINTAINER xuekui <kui.xue@dmall.com>

COPY bin/sym-admin-controller .
USER nonroot:nonroot

ENTRYPOINT ["/sym-admin-controller"]
