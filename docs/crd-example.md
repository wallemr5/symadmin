# CRD example for yaml

## AppSet

``` yaml
spec:
    labels:
        k1: v1
        k2: v2
    meta:
        k1: v1
        k2: v2
    totalReplicas: 20
    serviceName: nginx
    podSpec:
        selector: metav1.LabelSelector
        template:
            rawValues: ""
            chart:
                rawChart: XXXX # []byte
                url: http://xxxx.xxxx
            yaml: corev1.PodTemplateSpec
    clusterTopology:
    - zone:
        name: "北方地区"
        tag: "gz01"
        replicas: 10
        - cluster:
            name: "康盛机房"
            tag: "gz01a"
            replicas: 5
            template:
                rawValues: ""
            workload:
            - group:
                type: "Blue" # Blue/Green/Canary
                name: "aabb-gz01a-blue"
                replicas: 2
                nodeSelectorTerm: corev1.NodeSelectorTerm
                template:
                    rawValues: ""
                    chart: # you can specify this attribute for a special group setting.
                        rawChart: XXXX # []byte
                        url: http://xxxx.xxxx
                    yaml: corev1.PodTemplateSpec
            - group:
                type: "Green" # Blue/Green/Canary
                name: "aabb-gz01a-blue"
                replicas: 2
                nodeSelectorTerm: corev1.NodeSelectorTerm
                template:
                    rawValues: ""
            - group:
                type: "Canary" # Blue/Green/Canary
                name: "aabb-gz01a-blue"
                replicas: 1
                nodeSelectorTerm: corev1.NodeSelectorTerm
                template:
                    rawValues: ""                   
        - cluster:
            name: "腾讯云5区"
            tag: "gz01b"
            template:
                rawValues: ""
            workload:
            - group:
                type: "Green" # Blue/Green/Canary
                name: "aabb-gz01b-blue"
                replicas: 2
                nodeSelectorTerm: corev1.NodeSelectorTerm
                template:
                    rawValues: ""
    - zone:
        name: "西南地区"
        tag: "rz01"
        replicas: 10
        - cluster:
            name: "tke-cd1"
            tag: "rz01a"
            replicas: 5
            template:
                rawValues: ""
            workload:
            - group:
                type: "Blue" # Blue/Green/Canary
                name: "aabb-rz01a-blue"
                nodeSelectorTerm: corev1.NodeSelectorTerm
                replicas: 2
                template:
                    rawValues: ""
        - cluster:
            name: "tke-cd2"
            tag: "rz01b"
            template:
                rawValues: ""
            workload:
            - group:
                type: "Green" # Blue/Green/Canary
                name: "aabb-rz01b-blue"
                nodeSelectorTerm: corev1.NodeSelectorTerm
                replicas: 2
                rawValues: ""
status:
    observedGeneration: 1
    totalReplicas: 10
    totalAvailableCount: 5
    totalUnavailableCount: 2
    - zone:
        tag: "gz01"
        - cluster:
            tag: "gz01a"
            replicas: 10
            availableCount: 5
            unavailableCount: 2
            serviceType: "ClusterIP" # ClusterIP/NodePort/LoadBalancer
            clusterIP: ""
            domain: ""
            workload:
            - group:
                type: "Canary"
                replicas: 10
                availableCount: 5
                unavailableCount: 2
            - group:
                type: "Blue"
                replicas: 10
                availableCount: 5
                unavailableCount: 2
            - group:
                type: "Green"
                replicas: 10
                availableCount: 5
                unavailableCount: 2
            conditions:
            - type: "Available" # Available/Progressing/ReplicaFailure
              status: True # True/False/Unknown
              lastUpdateTime: "2019-12-26 13:37:47"
              lastTransitionTime: "2019-12-26 13:37:59"
              reason: ""
              message: ""
            - type: "Available" # Available/Progressing/ReplicaFailure
              status: True # True/False/Unknown
              lastUpdateTime: "2019-12-26 13:37:47"
              lastTransitionTime: "2019-12-26 13:37:59"
              reason: ""
              message: ""
              events: array
        - cluster:
            tag: "gz01b"
            replicas: 10
            availableCount: 5
            unavailableCount: 2
            serviceType: "ClusterIP" # ClusterIP/NodePort/LoadBalancer
            clusterIP: ""
            domain: ""
            workload:
            - group:
                type: "Canary"
                replicas: 10
                availableCount: 5
                unavailableCount: 2
            - group:
                type: "Blue"
                replicas: 10
                availableCount: 5
                unavailableCount: 2
            - group:
                type: "Green"
                replicas: 10
                availableCount: 5
                unavailableCount: 2
            conditions:
            - type: "Available" # Available/Progressing/ReplicaFailure
                status: True # True/False/Unknown
                lastUpdateTime: "2019-12-26 13:37:47"
                lastTransitionTime "2019-12-26 13:37:59"
                reason: ""
                message: ""
            - type: "Available" # Available/Progressing/ReplicaFailure
                status: True # True/False/Unknown
                lastUpdateTime: "2019-12-26 13:37:47"
                lastTransitionTime "2019-12-26 13:37:59"
                reason: ""
                message: ""
            message: ""
            events: array
    status: Running # Running/Migrating/WorkRatioing/Scaling/Updateing/Installing/Unknown
```

## AdvDeployment

``` yaml
spec:
    serviceName: nginx
    totalReplicas: 5
    podSpec:
        selector: metav1.LabelSelector
        template:
            rawValues: ""
            chart:
                rawChart: XXXX # []byte
                url: http://xxxx.xxxx
            yaml: corev1.PodTemplateSpec
    workload:
    - group:
        type: "Blue" # Blue/Green/Canary
        name: "aabb-gz01a-blue"
        replicas: 2
        nodeSelectorTerm: corev1.NodeSelectorTerm
        template:
            rawValues: ""
    - group:
        type: "Green" # Blue/Green/Canary
        name: "aabb-gz01a-blue"
        replicas: 2
        nodeSelectorTerm: corev1.NodeSelectorTerm
        template:
            rawValues: ""
    - group:
        type: "Canary" # Blue/Green/Canary
        name: "aabb-gz01a-blue"
        replicas: 1
        nodeSelectorTerm: corev1.NodeSelectorTerm
        template:
            rawValues: ""
status:
    version:
    - "v1"
    - "v2"
    message: ""
    replicas: 10
    availableCount: 5
    unavailableCount: 2
    serviceType: "ClusterIP" # ClusterIP/NodePort/LoadBalancer
    clusterIP: ""
    domain: ""
    workload:
    - group:
        type: "Canary"
        replicas: 10
        availableCount: 5
        unavailableCount: 2
    - group:
        type: "Blue"
        replicas: 10
        availableCount: 5
        unavailableCount: 2
    - group:
        type: "Green"
        replicas: 10
        availableCount: 5
        unavailableCount: 2
    conditions:
    - type: "Available" # Available/Progressing/ReplicaFailure
        status: True # True/False/Unknown
        lastUpdateTime: "2019-12-26 13:37:47"
        lastTransitionTime "2019-12-26 13:37:59"
        reason: ""
        message: ""
    - type: "Available" # Available/Progressing/ReplicaFailure
        status: True # True/False/Unknown
        lastUpdateTime: "2019-12-26 13:37:47"
        lastTransitionTime "2019-12-26 13:37:59"
        reason: ""
        message: ""
    observedGeneration: 1
    currentRevision: ""
    updateRevision: ""
    collisionCount: 1
    events: array
```


# nginx example

~~~yaml
apiVersion: workload.dmall.com/v1beta1
kind: AppSet
metadata:
  name: nginx
  namespace: default
spec:
  replicas: 10
  podSpec:
    deployType: helm
    chart:
      url: dmall/nginx
  serviceName: nginx-svc
  updateStrategy:
    upgradeType: canary|blue|green
    minReadySeconds: 10
    canaryClusters:
      - tcc-bj4-dks-test-01
    needWaitingForConfirm: true
    paused: false
  clusterTopology:
    - name: tcc-bj4-dks-test-01
      meta:
        area: beijing
        az: bj4
        ldc: TCC_BJ4_DKS_TEST_01
      podSets:
        - name: nginx-gz01b-canary
          replicas: 1
          version: v2
          meta:
            app: nginx
            version: v2
            sym-group: canary
            sym-ldc: gz01b
        - name: nginx-gz01b-blue
          replicas: 2
          version: v1
          meta:
            app: nginx
            version: v1
            sym-group: blue
            sym-ldc: gz01b
        - name: nginx-gz01b-green
          replicas: 2
          version: v1
          meta:
            app: nginx
            version: v1
            sym-group: green
            sym-ldc: gz01b
    - name: tcc-bj5-dks-test-01
      meta:
        area: beijing
        az: bj5
        ldc: TCC_BJ5_DKS_TEST_01
      podSets:
        - name: nginx-gz01a-canary
          replicas: 1
          version: v2
          meta:
            app: nginx
            version: v2
            sym-group: canary
            sym-ldc: gz01a
        - name: nginx-gz01a-blue
          replicas: 2
          version: v1
          meta:
            app: nginx
            version: v1
            sym-group: blue
            sym-ldc: gz01a
        - name: nginx-gz01a-green
          replicas: 2
          version: v1
          meta:
            app: nginx
            version: v1
            sym-group: green
            sym-ldc: gz01a
~~~