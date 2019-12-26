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
    replicas: 2
    serviceName: nginx
    deployType: helm # Helm/InPlaceSet/StatefulSet/Deployment
    podSpec:
        selector: metav1.LabelSelector
        template: corev1.PodTemplateSpec
        chart:
            rawChart: XXXX # []byte
            url: http://xxxx.xxxx
    updateStrategy:
        upgrageType: canary # canary/blue/green
        minReadySeconds: 30
        priorityStrategy:
            orderPriority:
                - orderdKey: ""
                - orderdKey: ""
            weightPriority:
                weight: 100
                matchSelector: metav1.LabelSelector
        canaryClusters:
        - ""
          ""
          ""
        paused: false
        needWaitingForConfirm: false
    clusterTopology:
        - clusters:
            name: ""
            podSets:
            - name: ""
              nodeSelectorTerm: corev1.NodeSelectorTerm
              replicas: 1
              rawValues: "" # use for helm
            _ name: ""
              nodeSelectorTerm: corev1.NodeSelectorTerm
              replicas: 1
              rawValues: "" # use for helm
status:
    observedGeneration: 1
    readyReplicas: 2
    replicas: 3
    updatedReplicas: 0
    updatedReadyReplicas: 2
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
    status: Running # Running/Migrating/WorkRatioing/Scaling/Updateing/Installing/Unknown
    appActual:
        total: 1
        items:
        - name: ""
          available: 1
          haveDeploy: false
          ready: 1
          update: 0
          current: 1
          running: 1
          warnEvent: 1
          endpointReady: 1
        - name: ""
          available: 1
          haveDeploy: false
          ready: 1
          update: 0
          current: 1
          running: 1
          warnEvent: 1
          endpointReady: 1
        pods:
        - name: ""
          namespace: ""
          state: ""
          podIP: ""
          nodeIP: ""
          clusterName: ""
          startTime: "2019-12-26 14:36:52"
        warnEvents:
        - message: ""
          sourceComponent: ""
          name: ""
          subObject: ""
          firstSeen: "2019-12-26 14:38:02"
          lastSeen: "2019-12-26 14:38:09"
          reason: ""
          type: ""
        - message: ""
          sourceComponent: ""
          name: ""
          subObject: ""
          firstSeen: "2019-12-26 14:38:02"
          lastSeen: "2019-12-26 14:38:09"
          reason: ""
          type: ""
        service:
            internalEndpoint
            labels:
            - k1: v1
              k2: v2
            selector:
            - k1: v1
              k2: v2
            type: "ClusterIP" # ClusterIP/NodePort/LoadBalancer
            clusterIP: ""
            Domain: ""
```

## AdvDeploymentSpec

``` yaml
spec:
    deployType: helm # helm/InPlaceSet/StatefulSet/deployment
    replicas: 1
    PodSpec:
        selector: metav1.LabelSelector
        template: corev1.PodTemplateSpec
        chart:
            rawChart: XXXX # []byte
            url: http://xxxx.xxxx
    serviceName: nginx
    updateStrategy:
        upgrageType: canary # canary/blue/green
        statefulSetStrategy:
            partition: 10
            maxUnavailable: 5 # 20%
            podUpdatePolicy: ReCreate # ReCreate/InPlaceIfPossible/InPlaceOnly
        minReadySeconds: 30
        meta:
            key1: value1
            key2: value2
        priorityStrategy:
            orderPriority:
                - orderdKey: ""
                - orderdKey: ""
            weightPriority:
                weight: 100
                matchSelector: metav1.LabelSelector
        paused: false
        needWaitingForConfirm: false
    topology:
        - podSets:
            name: ""
            nodeSelectorTerm: corev1.NodeSelectorTerm
            replicas: 1
            rawValues: "" # use for helm
        - podSets:
            name: ""
            nodeSelectorTerm: corev1.NodeSelectorTerm
            replicas: 1
            rawValues: "" # use for helm
    revisionHistoryLimit: 1 # default 10
status:
    version: 1.1
    message: ""
    replicas: 10
    readyReplicas: 5
    currentReplicas: 0
    updateReplicas: 0
    podSets:
    - name: nginx
        observedGeneration: 1
        replicas: 2
        readyReplicas: 1
        currentReplicas: 1
        updatedReplicas: 2
    - name: nginx
        observedGeneration: 1
        replicas: 2
        readyReplicas: 1
        currentReplicas: 1
        updatedReplicas: 2
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
```
