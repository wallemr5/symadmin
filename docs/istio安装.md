# 1.准备工作

~~~shell
# istio 下载  可能需要翻墙
$ curl -L https://git.io/getLatestIstio | ISTIO_VERSION=1.4.4 sh -

# 查看目录 
$ tree -L 2 istio-1.4.4
istio-1.4.4
├── LICENSE
├── README.md
├── bin
│   └── istioctl
├── install
│   ├── README.md
│   ├── consul
│   ├── gcp
│   ├── kubernetes
│   └── tools
├── manifest.yaml
├── samples
│   ├── README.md
│   ├── bookinfo
│   ├── certs
│   ├── custom-bootstrap
│   ├── external
│   ├── fortio
│   ├── health-check
│   ├── helloworld
│   ├── httpbin
│   ├── https
│   ├── kubernetes-blog
│   ├── multicluster
│   ├── operator
│   ├── rawvm
│   ├── security
│   ├── sleep
│   ├── tcp-echo
│   └── websockets
└── tools
    ├── _istioctl
    ├── convert_RbacConfig_to_ClusterRbacConfig.sh
    ├── dump_kubernetes.sh
    ├── istioctl.bash
    └── packaging

26 directories, 10 files

# istioctl 安装
$ cd istio-1.4.4
$ cp ./bin/istioctl /usr/local/bin

# istioctl 开启自动补全
# bash
$ cp tools/istioctl.bash ~/
$ source ~/istioctl.bash

# zsh
$ cp tools/_istioctl ~/
# 在~/.zsh最好添加一行 
source ~/_istioctl
# 应用生效
$ source ~/.zsh


# 修改kiali 中关于jaeger组件找不到问题
# 路径 install/kubernetes/helm/istio/charts/kiali/templates/configmap.yaml 原始内容：23行开始
    external_services:
      tracing:
        url: {{ .Values.dashboard.jaegerURL }}
      grafana:
        url: {{ .Values.dashboard.grafanaURL }}
      prometheus:
        url: {{ .Values.prometheusAddr }}
# 修改后内容
    external_services:
      tracing:
        namespace: {{ .Release.Namespace }}
        in_cluster_url: http://jaeger-query.istio-system.svc.cluster.local:16686
        url: {{ .Values.dashboard.jaegerURL }}
      grafana:
        in_cluster_url: http://grafana.istio-system.svc.cluster.local:3000
        url: {{ .Values.dashboard.grafanaURL }}
      prometheus:
        url: {{ .Values.prometheusAddr }}
        
# 修改后push到仓库
$ helm push install/kubernetes/helm/istio-cni dmall
$ helm push install/kubernetes/helm/istio-init dmall
$ helm push install/kubernetes/helm/istio dmall
~~~



# 2.单集群安装

采用initContainers注入iptables不用安装istio-cni

~~~shell
# istio-init values覆写文件
$ cat <<EOF > ./override-istio-init.yaml 
certmanager:
  enabled: true
EOF

# 安装
$ helm upgrade --install istio-init --namespace istio-system --debug -f override-istio-init.yaml dmall/istio-init

# 准备istio安装 values覆写文件
$cat <<EOF > ./override-istio-dkscd.yaml 
global:
  proxy:
    accessLogFile: "/dev/stdout"
    resources:
      requests:
        cpu: 10m
        memory: 40Mi

  disablePolicyChecks: false
  controlPlaneSecurityEnabled: false
  mtls:
    enabled: false
  arch:
    amd64: 2

sidecarInjectorWebhook:
  enabled: true
  rewriteAppHTTPProbe: false

pilot:
  autoscaleEnabled: false
  traceSampling: 100.0
  resources:
    requests:
      cpu: 10m
      memory: 100Mi

certmanager:
  enabled: true

mixer:
  policy:
    enabled: true
    autoscaleEnabled: false
    resources:
      requests:
        cpu: 10m
        memory: 100Mi

  telemetry:
    enabled: true
    autoscaleEnabled: false
    resources:
      requests:
        cpu: 50m
        memory: 100Mi

  adapters:
    stdio:
      enabled: true

prometheus:
  enabled: true
  contextPath: /
  image: prometheus
  tag: v2.15.2
  ingress:
    enabled: true
    hosts:
      - dkscdp.istio.dmall.com               # 定义prometheus访问url
    annotations:
      kubernetes.io/ingress.class: contour   # 注解指定ingress控制器  traefik/contour

grafana:
  enabled: true
  contextPath: /
  image:
    repository: grafana/grafana
    tag: 6.5.3
  ingress:
    enabled: true
    hosts:
      - dkscdg.istio.dmall.com
    annotations:
      kubernetes.io/ingress.class: traefik  # 注解指定ingress控制器  traefik/contour

tracing:
  enabled: true
  contextPath: /
  ingress:
    enabled: true
    hosts:
      - dkscdt.istio.dmall.com
    annotations:
      kubernetes.io/ingress.class: traefik   # 注解指定ingress控制器  traefik/contour

kiali:
  enabled: true
  contextPath: /
  createDemoSecret: true
  dashboard:
    auth:
      strategy: login
    secretName: kiali
    viewOnlyMode: false
    grafanaURL: http://dkscdg.istio.dmall.com # 定义kiali集群外能访问的 grafana url，集群内访问service域名已指定 
    jaegerURL: http://dkscdt.istio.dmall.com  # 定义kiali集群外能访问的 tracing url，集群内访问service域名已指定 
  ingress:
    enabled: true
    hosts:
      - dkscdk.istio.dmall.com
    annotations:
      kubernetes.io/ingress.class: traefik   # 注解指定ingress控制器  traefik/contour

gateways:
  istio-ingressgateway:
    serviceAnnotations:
      # 指定内网负载均衡注解，各种云实现方式不一样
      # tke找到集群网址子网id，<kubectl get svc -n default kube-user>  metadata.annotations 
      # aks下添加注解         service.beta.kubernetes.io/azure-load-balancer-internal: 'true'
      service.kubernetes.io/qcloud-loadbalancer-internal-subnetid: subnet-dch8t2h1
    autoscaleEnabled: false
    resources:
      requests:
        cpu: 10m
        memory: 40Mi

  istio-egressgateway:
    enabled: true
    autoscaleEnabled: false
    resources:
      requests:
        cpu: 10m
        memory: 40Mi
EOF

# 安装 istio
$ helm upgrade --install istio --namespace istio-system -f override-istio-dkscd dmall/istio
~~~

