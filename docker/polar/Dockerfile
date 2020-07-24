FROM symcn.tencentcloudcr.com/symcn/centos-base:v7.8.2

RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime  \
    && echo "Asia/Shanghai">>/etc/timezone

COPY polar  /

ENTRYPOINT ["/polar"]
