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
    replicas: 20
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
    - zoneInfo:
        zoneName: "北方地区"
        zoneTag: "gz01"
        zoneReplicas: 10
        - clusterInfo:
            clusterName: "康盛机房"
            clusterTag: "gz01a"
            clusterReplicas: 5
            template:
                rawValues: ""
                chart:
                    rawChart: XXXX # []byte
                    url: http://xxxx.xxxx
                yaml: corev1.PodTemplateSpec
            advInfo:
            - groupInfo:
                groupType: "Blue" # Blue/Green/Canary
                groupName: "aabb-gz01a-blue"
                groupReplicas: 2
                nodeSelectorTerm: corev1.NodeSelectorTerm
                template:
                    rawValues: ""
                    chart:
                        rawChart: XXXX # []byte
                        url: http://xxxx.xxxx
                    yaml: corev1.PodTemplateSpec
            - groupInfo:
                groupType: "Green" # Blue/Green/Canary
                groupName: "aabb-gz01a-blue"
                groupReplicas: 2
                nodeSelectorTerm: corev1.NodeSelectorTerm
                template:
                    rawValues: ""
                    chart:
                        rawChart: XXXX # []byte
                        url: http://xxxx.xxxx
                    yaml: corev1.PodTemplateSpec
            - groupInfo:
                groupType: "Canary" # Blue/Green/Canary
                groupName: "aabb-gz01a-blue"
                groupReplicas: 1
                nodeSelectorTerm: corev1.NodeSelectorTerm
                template:
                    rawValues: ""
                    chart:
                        rawChart: XXXX # []byte
                        url: http://xxxx.xxxx
                    yaml: corev1.PodTemplateSpec
        - clusterInfo:
            clusterName: "腾讯云5区"
            clusterTag: "gz01b"
            template:
                rawValues: ""
                chart:
                    rawChart: XXXX # []byte
                    url: http://xxxx.xxxx
                yaml: corev1.PodTemplateSpec
            advInfo:
            - groupInfo:
                groupType: "Green" # Blue/Green/Canary
                groupName: "aabb-gz01b-blue"
                groupReplicas: 2
                nodeSelectorTerm: corev1.NodeSelectorTerm
                template:
                    rawValues: ""
                    chart:
                        rawChart: XXXX # []byte
                        url: http://xxxx.xxxx
                    yaml: corev1.PodTemplateSpec
    - zoneInfo:
        zoneName: "西南地区"
        zoneTag: "rz01"
        zoneReplicas: 10
        - clusterInfo:
            clusterName: "tke-cd1"
            clusterTag: "rz01a"
            clusterReplicas: 5
            template:
                rawValues: ""
                chart:
                    rawChart: XXXX # []byte
                    url: http://xxxx.xxxx
                yaml: corev1.PodTemplateSpec
            advInfo:
            - groupInfo:
                groupType: "Blue" # Blue/Green/Canary
                groupName: "aabb-rz01a-blue"
                nodeSelectorTerm: corev1.NodeSelectorTerm
                groupReplicas: 2
                template:
                    rawValues: ""
                    chart:
                        rawChart: XXXX # []byte
                        url: http://xxxx.xxxx
        - clusterInfo:
            clusterName: "tke-cd2"
            clusterTag: "rz01b"
            template:
                rawValues: ""
                chart:
                    rawChart: XXXX # []byte
                    url: http://xxxx.xxxx
                yaml: corev1.PodTemplateSpec
            advInfo:
            - groupInfo:
                groupType: "Green" # Blue/Green/Canary
                groupName: "aabb-rz01b-blue"
                nodeSelectorTerm: corev1.NodeSelectorTerm
                groupReplicas: 2
                rawValues: ""
                chart:
                    rawChart: XXXX # []byte
                    url: http://xxxx.xxxx
status:
    observedGeneration: 1
    totalReplicas: 10
    totalAvailableCount: 5
    totalUnavailableCount: 2
    - zoneInfo:
        zoneTag: "gz01"
        - clusterInfo:
            clusterTag: "gz01a"
            clusterReplicas: 10
            clusterAvailableCount: 5
            clusterUnavailableCount: 2
            serviceType: "ClusterIP" # ClusterIP/NodePort/LoadBalancer
            clusterIP: ""
            domain: ""
            - group:
                groupType: "Canary"
                groupReplicas: 10
                groupAvailableCount: 5
                groupUnavailableCount: 2
            - group:
                groupType: "Blue"
                groupReplicas: 10
                groupAvailableCount: 5
                groupUnavailableCount: 2
            - group:
                groupType: "Green"
                groupReplicas: 10
                groupAvailableCount: 5
                groupUnavailableCount: 2
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
        - clusterInfo:
            clusterTag: "gz01b"
            clusterReplicas: 10
            clusterAvailableCount: 5
            clusterUnavailableCount: 2
            serviceType: "ClusterIP" # ClusterIP/NodePort/LoadBalancer
            clusterIP: ""
            domain: ""
            - group:
                groupType: "Canary"
                groupReplicas: 10
                groupAvailableCount: 5
                groupUnavailableCount: 2
            - group:
                groupType: "Blue"
                groupReplicas: 10
                groupAvailableCount: 5
                groupUnavailableCount: 2
            - group:
                groupType: "Green"
                groupReplicas: 10
                groupAvailableCount: 5
                groupUnavailableCount: 2
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
    clusterReplicas: 5
    podSpec:
        selector: metav1.LabelSelector
        template:
            rawValues: ""
            chart:
                rawChart: XXXX # []byte
                url: http://xxxx.xxxx
            yaml: corev1.PodTemplateSpec
    details:
    - groupInfo:
        groupType: "Blue" # Blue/Green/Canary
        groupName: "aabb-gz01a-blue"
        groupReplicas: 2
        nodeSelectorTerm: corev1.NodeSelectorTerm
        template:
            rawValues: ""
            chart:
                rawChart: XXXX # []byte
                url: http://xxxx.xxxx
            yaml: corev1.PodTemplateSpec
    - groupInfo:
        groupType: "Green" # Blue/Green/Canary
        groupName: "aabb-gz01a-blue"
        groupReplicas: 2
        nodeSelectorTerm: corev1.NodeSelectorTerm
        template:
            rawValues: ""
            chart:
                rawChart: XXXX # []byte
                url: http://xxxx.xxxx
            yaml: corev1.PodTemplateSpec
    - groupInfo:
        groupType: "Canary" # Blue/Green/Canary
        groupName: "aabb-gz01a-blue"
        groupReplicas: 1
        nodeSelectorTerm: corev1.NodeSelectorTerm
        template:
            rawValues: ""
            chart:
                rawChart: XXXX # []byte
                url: http://xxxx.xxxx
            yaml: corev1.PodTemplateSpec
status:
    version:
    - "v1"
    - "v2"
    message: ""
    clusterReplicas: 10
    clusterAvailableCount: 5
    clusterUnavailableCount: 2
    serviceType: "ClusterIP" # ClusterIP/NodePort/LoadBalancer
    clusterIP: ""
    domain: ""
    details:
    - groupInfo:
        groupType: "Canary"
        groupReplicas: 10
        groupAvailableCount: 5
        groupUnavailableCount: 2
    - groupInfo:
        groupType: "Green"
        groupReplicas: 10
        groupAvailableCount: 5
        groupUnavailableCount: 2
    - groupInfo:
        groupType: "Blue"
        groupReplicas: 10
        groupAvailableCount: 5
        groupUnavailableCount: 2
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
