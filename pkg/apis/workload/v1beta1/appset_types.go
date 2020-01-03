package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&AppSet{}, &AppSetList{})
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// +kubebuilder:object:root=true
// AppSet represents a union for app

// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=as
// +kubebuilder:printcolumn:name="DESIRED",type="integer",JSONPath=".spec.desired",description="The desired number of pods."
// +kubebuilder:printcolumn:name="AVAILABEL",type="integer",JSONPath=".status.available",description="The number of pods ready."
// +kubebuilder:printcolumn:name="VERSION",type="string",JSONPath=".status.version",description="The image version."
// +kubebuilder:printcolumn:name="STATUS",type="string",JSONPath=".status.status",description="The app run status."
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp",description="CreationTimestamp is a timestamp representing the server time when this object was created. "
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

	// template is the object that describes the pod that will be created if
	// insufficient replicas are detected. Each pod stamped out by the workload
	// will fulfill this Template, but have a unique identity from the rest
	// of the workload.
	PodSpec PodSpec `json:"podSpec,omitempty"`

	// UpdateStrategy indicates the strategy the advDeployment use to preform the update,
	// when template is changed.
	// +optional
	UpdateStrategy AppSetUpdateStrategy `json:"updateStrategy,omitempty"`
	// Topology describes the pods distribution detail between each of subsets.
	// +optional
	ClusterTopology ClusterTopology `json:"clusterTopology,omitempty"`
}

type AppSetUpdateStrategy struct {
	// canary, blue, green
	UpgradeType           string                  `json:"upgradeType,omitempty"`
	MinReadySeconds       int32                   `json:"minReadySeconds,omitempty"`
	PriorityStrategy      *UpdatePriorityStrategy `json:"priorityStrategy,omitempty"`
	CanaryClusters        []string                `json:"canaryClusters,omitempty"`
	Paused                bool                    `json:"paused,omitempty"`
	NeedWaitingForConfirm bool                    `json:"needWaitingForConfirm,omitempty"`
}

type ClusterTopology struct {
	Clusters []*TargetCluster `json:"clusters,omitempty"`
}

type TargetCluster struct {
	// Target cluster name
	Name string `json:"name,omitempty"`

	// exp: zone, rack
	Mata map[string]string `json:"meta,omitempty"`

	// Contains the details of each subset. Each element in this array represents one subset
	// which will be provisioned and managed by UnitedDeployment.
	// +optional
	PodSets []*PodSet `json:"podSets,omitempty"`
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

	StartTime      *metav1.Time `json:"startTime,omitempty"`
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	//
	Version string `json:"version,omitempty"`

	//
	Desired int32 `json:"desired,omitempty"`

	//
	Available int32 `json:"available,omitempty"`

	// Represents the latest available observations of a UnitedDeployment's current state.
	// +optional
	Conditions []AppSetCondition `json:"conditions,omitempty"`
	Status     AppSatus          `json:"status,omitempty"`
	AppActual  AppActual         `json:"appActual,omitempty"`
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

// ClusterAppActual
type ClusterAppActual struct {
	Desired   int32             `json:"desired"`
	Available int32             `json:"available"`
	PodSets   []PodSetSatusInfo `json:"podSets,omitempty"`
}

// AppActual represent the app status
type AppActual struct {
	Clusters   []*ClusterAppActual `json:"clusters,omitempty"`
	Pods       []*Pod              `json:"pods,omitempty"`
	WarnEvents []*Event            `json:"warnEvents,omitempty"`
	Service    *Service            `json:"service,omitempty"`
}
