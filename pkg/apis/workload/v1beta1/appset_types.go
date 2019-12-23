package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// AppSet represents a union for app
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AppSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppSetSpec   `json:"spec,omitempty"`
	Status AppSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// AppStatusList implements list of AppStatus.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AppSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppSet `json:"items"`
}

// AppSetSpec contains AppSet specification
type AppSetSpec struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Meta        map[string]string `json:"meta,omitempty"`
	Replicas    *int32            `json:"replicas,omitempty"`
	ServiceName *string           `json:"serviceName,omitempty"`

	// support PodSet：helm, InPlaceSet，StatefulSet, deployment
	// Default value is deployment
	// +optional
	DeployType string `json:"deployType,omitempty"`
	// template is the object that describes the pod that will be created if
	// insufficient replicas are detected. Each pod stamped out by the workload
	// will fulfill this Template, but have a unique identity from the rest
	// of the workload.
	PodSpec PodSpec `json:",inline"`

	// Topology describes the pods distribution detail between each of subsets.
	// +optional
	ClusterTopology ClusterTopology `json:"clusterTopology,omitempty"`
}

type ClusterTopology struct {
	// Contains the details of each subset. Each element in this array represents one subset
	// which will be provisioned and managed by UnitedDeployment.
	// +optional
	PodSets map[string][]PodSet `json:"podSets,omitempty"`
}

type TargetCluster struct {
	// Target cluster name
	Name string `json:"name,omitempty"`
}

// AppSetConditionType indicates valid conditions type of a UnitedDeployment.
type AppSetConditionType string

// UnitedDeploymentCondition describes current state of a UnitedDeployment.
type AppSetCondition struct {
	// Type of in place set condition.
	Type AppSetConditionType `json:"type,omitempty"`

	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status,omitempty"`

	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`

	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

// AppSetStatus contains AppSet status
type AppSetStatus struct {
	// ObservedGeneration is the most recent generation observed for this worklod. It corresponds to the
	// worklod's generation, which is updated on mutation by the API Server.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
	// The number of ready replicas.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Replicas is the most recently observed number of replicas.
	Replicas int32 `json:"replicas"`

	// The number of pods in current version.
	UpdatedReplicas int32 `json:"updatedReplicas"`

	// The number of ready current revision replicas for this UnitedDeployment.
	// +optional
	UpdatedReadyReplicas int32 `json:"updatedReadyReplicas,omitempty"`
	// Represents the latest available observations of a UnitedDeployment's current state.
	// +optional
	Conditions []AppSetCondition `json:"conditions,omitempty"`
	Status     AppSatus          `json:"status,omitempty"`
	AppAggr
}

// app status
type AppSatus string

const (
	AppSatusRuning       AppSatus = "Running"
	AppSatusMigrating    AppSatus = "Migrating"
	AppSatusWorkRatioing AppSatus = "WorkRatioing"
	AppSatusScaling      AppSatus = "Scaling"
	AppSatusUpdateing    AppSatus = "Updateing"
	AppSatusInstalling   AppSatus = "Installing"
	AppSatusUnknown      AppSatus = "Unknown"
)

// type AppSetConditionType string
//
// type AppSetCondition struct {
// 	Type               AppSetConditionType    `json:"type"`
// 	Status             kapiv1.ConditionStatus `json:"status"`
// 	LastProbeTime      metav1.Time            `json:"lastProbeTime,omitempty"`
// 	LastTransitionTime metav1.Time            `json:"lastTransitionTime,omitempty"`
// 	Reason             string                 `json:"reason,omitempty"`
// 	Message            string                 `json:"message,omitempty"`
// }

type AppAggr struct {
	Desired    AppDesired `json:"desired"`
	Actual     AppActual  `json:"actual"`
	HaveDeploy bool       `json:"haveDeploy"`
}

type AppDesiredItem struct {
	Name       string `json:"Name"`
	Desired    int32  `json:"desired"`
	HaveDeploy bool   `json:"haveDeploy"`
}

type AppDesired struct {
	Total int32             `json:"total"`
	Items []*AppDesiredItem `json:"items,omitempty"`
}

// AppActualItems
type AppActualItem struct {
	Name          string `json:"Name"`
	Available     int32  `json:"available"`
	HaveDeploy    bool   `json:"haveDeploy"`
	Ready         *int32 `json:"ready,omitempty"`
	Update        *int32 `json:"update,omitempty"`
	Current       *int32 `json:"current,omitempty"`
	Running       *int32 `json:"running,omitempty"`
	WarnEvent     *int32 `json:"warnEvent,omitempty"`
	EndpointReady *int32 `json:"endpointReady,omitempty"`
}

// AppActual represent the app status
type AppActual struct {
	Total      int32            `json:"total"`
	Items      []*AppActualItem `json:"items,omitempty"`
	Pods       []*Pod           `json:"pods,omitempty"`
	WarnEvents []*Event         `json:"warnEvents,omitempty"`
	Service    *Service         `json:"service,omitempty"`
}

func DeepEqualAppDesired(x, y AppDesired) bool {
	if x.Total != y.Total {
		return false
	}

	if len(x.Items) != len(y.Items) {
		return false
	}

	for i := range x.Items {
		if x.Items[i].Name != y.Items[i].Name {
			return false
		}

		if x.Items[i].Desired != y.Items[i].Desired {
			return false
		}
	}
	return true
}

func DeepEqualAppActual(x, y AppActual) bool {
	if x.Total != y.Total {
		return false
	}

	if len(x.Items) != len(y.Items) {
		return false
	}

	for i := range x.Items {
		if x.Items[i].Name != y.Items[i].Name {
			return false
		}

		if x.Items[i].Available != y.Items[i].Available {
			return false
		}
	}

	if len(x.Pods) != len(y.Pods) {
		return false
	}

	for i := range x.Pods {
		if x.Pods[i].Name != y.Pods[i].Name {
			return false
		}

		if x.Pods[i].Namespace != y.Pods[i].Namespace {
			return false
		}

		if x.Pods[i].State != y.Pods[i].State {
			return false
		}

		if x.Pods[i].PodIp != y.Pods[i].PodIp {
			return false
		}

		if x.Pods[i].NodeName != y.Pods[i].NodeName {
			return false
		}

		if x.Pods[i].ClusterName != y.Pods[i].ClusterName {
			return false
		}

		if x.Pods[i].StartTime.Second() != y.Pods[i].StartTime.Second() {
			return false
		}
	}

	if len(x.WarnEvents) != len(y.WarnEvents) {
		return false
	}

	for i := range x.WarnEvents {
		if x.WarnEvents[i].Name != y.WarnEvents[i].Name {
			return false
		}

		if x.WarnEvents[i].Message != y.WarnEvents[i].Message {
			return false
		}
	}

	if x.Service != nil && y.Service != nil {
		if x.Service.Type != y.Service.Type {
			return false
		}
		if x.Service.ClusterIP != y.Service.ClusterIP {
			return false
		}

	} else {
		return false
	}
	return true
}

func DeepEqualAppAggr(x, y *AppAggr) bool {
	if !DeepEqualAppDesired(x.Desired, y.Desired) {
		return false
	}

	if !DeepEqualAppActual(x.Actual, y.Actual) {
		return false
	}

	return true
}
