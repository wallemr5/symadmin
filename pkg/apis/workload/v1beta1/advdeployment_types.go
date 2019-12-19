/*
Copyright 2019 The dks authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
)

// PodUpdateStrategyType is a string enumeration type that enumerates
// all possible ways we can update a Pod when updating application
type PodUpdateStrategyType string

const (
	RecreatePodUpdateStrategyType          PodUpdateStrategyType = "ReCreate"
	InPlaceIfPossiblePodUpdateStrategyType PodUpdateStrategyType = "InPlaceIfPossible"
	InPlaceOnlyPodUpdateStrategyType       PodUpdateStrategyType = "InPlaceOnly"
)

// StatefulSetStrategy is used to communicate parameter for StatefulSetStrategyType.
type StatefulSetStrategy struct {
	Partition       *int32                `json:"partition,omitempty"`
	MaxUnavailable  *intstr.IntOrString   `json:"maxUnavailable,omitempty"`
	PodUpdatePolicy PodUpdateStrategyType `json:"podUpdatePolicy,omitempty"`
}

type UpdateStrategy struct {
	// Beta, Batch, BlueGreen, Cell
	UpgradeType           string               `json:"upgradeType,omitempty"`
	BatchSize             *int32               `json:"batchSize,omitempty"`
	RzNum                 *int32               `json:"rzNum,omitempty"`
	StatefulSetStrategy   *StatefulSetStrategy `json:"statefulSetStrategy,omitempty"`
	Paused                bool                 `json:"paused,omitempty"`
	NeedWaitingForConfirm bool                 `json:"needWaitingForConfirm,omitempty"`
	MinReadySeconds       int32                `json:"minReadySeconds,omitempty"`
	CellReplicas          []*CellReplicas      `json:"cellReplicas,omitempty"`
	Meta                  map[string]string    `json:"meta,omitempty"`
}

type CellReplicas struct {
	CellName string `json:"cellName,omitempty"`
	Replicas int32  `json:"replicas,omitempty"`
}

type ClusterAllocator struct {
	Name        string            `json:"name"`
	AllocFactor int               `json:"allocFactor"`
	Meta        map[string]string `json:"meta,omitempty"`
}
type ClusterRef struct {
	ClusterInfoRef    *v1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`
	ClusterAllocators []*ClusterAllocator      `json:"clusterAllocators,omitempty"`
}

// AdvDeploymentSpec defines the desired state of AdvDeployment
type AdvDeploymentSpec struct {
	// support PodSet：InPlaceSet，StatefulSet, deployment
	// Default value is deployment
	WorkloadType         string                     `json:"workloadType,omitempty"`
	RevisionHistoryLimit *int32                     `json:"revisionHistoryLimit,omitempty"`
	Replicas             *int32                     `json:"replicas,omitempty"`
	Domain               *string                    `json:"domain,omitempty"`
	Selector             *metav1.LabelSelector      `json:"selector,omitempty"`
	Template             v1.PodTemplateSpec         `json:"template"`
	VolumeClaimTemplates []v1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`
	ServiceName          string                     `json:"serviceName,omitempty"`
	Strategy             UpdateStrategy             `json:"strategy,omitempty"`
	InstallMultiClusters bool                       `json:"installMultiClusters,omitempty"`
	ClusterRef           *ClusterRef                `json:"clusterRef,omitempty"`
}

type AdvDeploymentConditionType string

// These are valid conditions of a deployment.
const (
	// Available means the deployment is available, ie. at least the minimum available
	// replicas required are up and running for at least minReadySeconds.
	DeploymentAvailable AdvDeploymentConditionType = "Available"
	// Progressing means the deployment is progressing. Progress for a deployment is
	// considered when a new replica set is created or adopted, and when new pods scale
	// up or old pods scale down. Progress is not estimated for paused deployments or
	// when progressDeadlineSeconds is not specified.
	DeploymentProgressing AdvDeploymentConditionType = "Progressing"
	// ReplicaFailure is added in a deployment when one of its pods fails to be created
	// or deleted.
	DeploymentReplicaFailure AdvDeploymentConditionType = "ReplicaFailure"
)

// AdvDeploymentCondition describes the state of a adv deployment at a certain point.
type AdvDeploymentCondition struct {
	// Type of deployment condition.
	Type AdvDeploymentConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty"`
}

type PodSetStatus struct {
	Name                string `json:"name,omitempty"`
	ObservedGeneration  int64  `json:"observedGeneration,omitempty"`
	Replicas            int32  `json:"replicas,omitempty"`
	UpdatedReplicas     int32  `json:"updatedReplicas,omitempty"`
	ReadyReplicas       int32  `json:"readyReplicas,omitempty"`
	AvailableReplicas   int32  `json:"availableReplicas,omitempty"`
	UnavailableReplicas int32  `json:"unavailableReplicas,omitempty"`
}

type DeployState string

const (
	Created         DeployState = "Created"
	ReconcileFailed DeployState = "ReconcileFailed"
	Reconciling     DeployState = "Reconciling"
	Available       DeployState = "Available"
	Unmanaged       DeployState = "Unmanaged"
)

// AdvDeploymentStatus defines the observed state of AdvDeployment
type AdvDeploymentStatus struct {
	Status        DeployState              `json:"status,omitempty"`
	Version       string                   `json:"version,omitempty"`
	Message       string                   `json:"message,omitempty"`
	Replicas      int32                    `json:"replicas,omitempty" `
	ReadyReplicas int32                    `json:"readyReplicas,omitempty" `
	PodSets       map[string]PodSetStatus  `json:"podSets,omitempty"`
	Conditions    []AdvDeploymentCondition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true

// AdvDeployment is the Schema for the advdeployments API
type AdvDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AdvDeploymentSpec   `json:"spec,omitempty"`
	Status AdvDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AdvDeploymentList contains a list of AdvDeployment
type AdvDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AdvDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AdvDeployment{}, &AdvDeploymentList{})
}

// Default makes AdvDeployment an mutating webhook
// When delete, if error occurs, finalizer is a good options for us to retry and
// record the events.
func (in *AdvDeployment) Default() {
	if !in.DeletionTimestamp.IsZero() {
		return
	}

	klog.V(4).Info("AdvDeployment: ", in.GetName())
}

// ValidateCreate implements webhook.Validator
// 1. check filed regex
func (in *AdvDeployment) ValidateCreate() error {
	klog.V(4).Info("validate AdvDeployment create: ", in.GetName())

	return nil
}

// ValidateUpdate validate HelmRequest update request
// immutable fields:
// 1. ...
func (in *AdvDeployment) ValidateUpdate(old runtime.Object) error {
	klog.V(4).Info("validate HelmRequest update: ", in.GetName())

	oldHR, ok := old.(*AdvDeployment)
	if !ok {
		return fmt.Errorf("expect old object to be a %T instead of %T", oldHR, old)
	}

	return nil
}
