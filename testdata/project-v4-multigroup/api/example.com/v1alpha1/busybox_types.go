/*
Copyright 2025 The Kubernetes authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BusyboxSpec defines the desired state of Busybox
type BusyboxSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// size defines the number of Busybox instances
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1
	// +required
	Size int32 `json:"size"`
}

// BusyboxStatus defines the observed state of Busybox.
// For guidance, see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
type BusyboxStatus struct {
	// conditions represent the latest available observations of the resource's state.
	// Common condition types include:
	// - "Available": the resource is healthy and functioning as expected.
	// - "Progressing": the resource is being created or updated.
	// - "Degraded": the resource has encountered an error or failed to reach its desired state.
	//
	// Each condition must have a unique type. A condition's 'status' field may be True, False, or Unknown.
	//
	// +optional
	// +patchStrategy=merge
	// +patchMergeKey=type
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Busybox is the Schema for the busyboxes API
type Busybox struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata.
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the desired state of Busybox.
	// +required
	Spec BusyboxSpec `json:"spec"`

	// status defines the observed state of Busybox.
	// +optional
	Status *BusyboxStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BusyboxList contains a list of Busybox
type BusyboxList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Busybox `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Busybox{}, &BusyboxList{})
}
