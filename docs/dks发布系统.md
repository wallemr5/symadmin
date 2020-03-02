# 架构



# 概述



# 设计

## 声明式

<u>**声明式**</u>（Declarative）的编程是云原生的一大特色，与<u>**命令式编程**</u>相比，能更好的描述一个效果或者目标的终态。

在 Kubernetes 中，我们可以直接使用 YAML 文件定义应用服务在**<u>多集群的拓扑结构和状态</u>**：

~~~yaml
apiVersion: workload.dmall.com/v1beta1
kind: AppSet
metadata:
  name: bbcc
  namespace: default
spec:
  replicas: 10
  podSpec:
    deployType: helm                 # helm, InPlaceSet，StatefulSet, deployment等， 目前支持helm
    chart:                           # helm部署时生效
      chartUrl:                      # helm仓库chart描述
        url: dmall/springBoot        
        chartVersion: 0.0.1
      rawChart: ...                  # 没有helm仓库时可指定原始压缩包  []byte
    template: ...                    # 其他部署方式是pod的原始描述模板 PodTemplateSpec
  serviceName: inner.bbcc.dmall.com  # App 的 service 名称
  updateStrategy:                    # 升级策略
    upgradeType: canary|blue|green   
    minReadySeconds: 10
    canaryClusters:                  # 灰度部署集群拓扑
      - tcc-bj4-dks-test-01
    needWaitingForConfirm: true
    paused: false
  clusterTopology:
    clusters:
      - name: tcc-bj4-dks-test-01
        meta:
          sym-available-zone: bj4
          sym-cluster-info: tcc-bj4-dks-test-01
        podSets:
          - name: bbcc-gz01b-canary
            replicas: 1
            version: v4
          - name: bbcc-gz01b-blue
            replicas: 2
            version: v3
          - name: bbcc-gz01b-green
            replicas: 2      # override replicas
            version: v3      # override version
            image: ...       # override image
            chart: ...       # override chart helm部署时生效
            rawValues: ...   # override rawValues  helm部署时生效
            meta:
              sym-group: green
      - name: tcc-bj5-dks-test-01
        meta:
          sym-available-zone: bj5
          sym-cluster-info: tcc-bj5-dks-test-01
        podSets:
          - name: bbcc-gz01a-canary
            replicas: 1
            version: v4
          - name: bbcc-gz01a-blue
            replicas: 2
            version: v3
          - name: bbcc-gz01a-green
            replicas: 2
            version: v3
~~~

部署完成后可以看到部署的状态：

~~~shell
$ kubectl get as --all-namespaces
NAMESPACE     NAME                      DESIRED   AVAILABLE   UNAVAILABLE   VERSION   STATUS    AGE
dmall-inner   kafka-08-producer-gz01a   6         6           0             v8/v9     Running   3h37m
dmall-inner   kafka-test-group          8         8           0             v2        Running   54m
dmall-inner   no-project-aabb           10        10          0             v5        Running   47h

$ kubectl get ad --all-namespaces
NAMESPACE     NAME                      DESIRED   AVAILABLE   UNAVAILABLE   VERSION   STATUS    AGE
dmall-inner   kafka-08-producer-gz01a   2         2           0             v8/v9     Running   3h39m
dmall-inner   kafka-test-group          4         4           0             v2        Running   56m
dmall-inner   no-project-aabb           4         4           0             v5        Running   47h
~~~

## 控制器模式

控制器采用松耦合的方式组合，启动命令：

~~~shell
# 只启动 AppSetController 控制器
$ sym-controller controller --enable-master -v 4   

 # 只启动 AdvDeploymentController 控制器
$ sym-controller controller --enable-worker -v 4   

# 同时启动 AppSetController 和 AdvDeploymentController 控制器
$ sym-controller controller --enable-master --enable-worker -v 4  
~~~

由于在多集群中部署，各个资源无法建立ownerReferences关系，也就无法控制从属资源的生命周期；在此基础上引入了Finalizers机制，用户发起删除资源后，控制器查询Finalizers是否为空，待从属资源AdvDeployment、helm release等删除后，清空Finalizers资源也立即被删除。

### 实现原理

- 调谐，AppSetController作为管理 AppSet 资源的控制器，会在启动时通过 `Informer` 监听多集群两种不同资源的通知，AppSet和 AdvDeployment，这些资源的变动都会触发 `AppSetController` 中的回调。根据集群拓扑调谐相应集群的AdvDeployment资源。

- 状态收集，触发回调后，获取AdvDeployment资源状态然后统一聚合。

### 多集群管理

- 集群感知，通过通过 `Informer` 监听元集群configmap资源变化，获取配置后初始化客户端，可以运行后动态增加删除基础，修改元数据。
- 创建一些额外的资源索引后开启Informer cache同步协程，等待资源同步完成。
- 同时开启一个后台协程负责集群健康检查，根据一定策略删除和恢复集群管理。

### 健康检查

~~~go
// Check is a health/readiness check.
type Check func() error

// Handler is an endpoints with additional methods that register health and
// readiness checks. It handles handle "/live" and "/ready" HTTP
// endpoints.
type Handler interface {
	Routes() []*router.Route
	AddLivenessCheck(name string, check Check)
	AddReadinessCheck(name string, check Check)
	LiveEndpoint(ctx *gin.Context)
	ReadyEndpoint(ctx *gin.Context)
	RemoveLivenessCheck(name string)
	RemoveReadinessCheck(name string)
}
~~~



## api组件

资源获取采用标准http接口提供，启动命令：

~~~shell
# 集群内部部署
$ sym-api api -v 4

# 集群外部部署,kubeconfig指向master集群
$ sym-api api --kubeconfig=./manifests/kubeconfig_TCC_BJ5_DKS_MONIT_01.yaml -v 4
~~~



