FROM centos:7.7.1908

MAINTAINER xk <kui.xue@dmall.com>

RUN yum update -y && yum install -y \
    deltarpm  \
    device-mapper \
    bridge-utils \
    ipvsadm \
    iptables \
    lsof \
    curl \
    file \
    gdb \
    git \
    iotop \
    linux-perf \
    mysql \
    net-tools \
    perf \
    perl \
    procps-ng \
    psmisc \
    strace \
    sysstat \
    tree \
    tcpdump \
    unzip \
    vim \
    wget \
    which \
    netstat \
    tcptraceroute \
    && yum clean all \
    && rm -rf /var/cache/yum/*


# used for go pprof
ENV GOLANG_VERSION 1.13.5
ENV GOLANG_DOWNLOAD_URL https://studygolang.com/dl/golang/go$GOLANG_VERSION.linux-amd64.tar.gz
ENV GOLANG_DOWNLOAD_SHA256 512103d7ad296467814a6e3f635631bd35574cab3369a97a323c9a585ccaa569
RUN curl -fsSL "$GOLANG_DOWNLOAD_URL" -o golang.tar.gz \
    && echo "$GOLANG_DOWNLOAD_SHA256  golang.tar.gz" | sha256sum -c - \
    && tar -C /usr/local -xzf golang.tar.gz \
    && rm golang.tar.gz
ENV GOPATH /go
ENV GOROOT /usr/local/go
ENV PATH $GOPATH/bin:$GOROOT/bin:$PATH

COPY Dockerfile /Dockerfile

ENTRYPOINT ["/bin/bash", "-l"]
