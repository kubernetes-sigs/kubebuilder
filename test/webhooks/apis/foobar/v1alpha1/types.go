/*
Copyright 2018 The Kubernetes Authors.

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
	"encoding/json"
	"github.com/kubernetes-sigs/kubebuilder/pkg/webhooks/"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const Kindz = "FooBar"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FooBar is a test object for webhooks.
type FooBar struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec holds the desired state of the FooBar (from the client).
	Spec FooBarSpec `json:"spec,omitempty"`

	// Status communicates the observed state of the FooBar (from the controller).
	Status FooBarStatus `json:"status,omitempty"`
}

// FooBarSpec holds the desired state of the FooBar (from the client).
type FooBarSpec struct {
	// Foos is a count of foos requested.
	Foos int64 `json:"foos,omitempty"`

	// Bars is a count of bars requested.
	Bars int64 `json:"bars,omitempty"`
}

type FooBarCondition struct {
	Type string `json:"type" description:"type of condition"`

	Status corev1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" description:"last time the condition transit from one status to another"`

	// +optional
	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
}

// FooBarStatus communicates the observed state of the FooBar (from the controller).
type FooBarStatus struct {
	// Conditions communicates information about ongoing/complete reconciliation processes that brings the "spec" inline
	// with the observed state.
	Conditions []FooBarCondition `json:"conditions,omitempty"`

	// ObservedGeneration is the 'Generation' of the resource that was last processed by the controller. The
	// observed generation is updated even if the controller failed to process the spec and create the FooBar.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FooBarList is a list of FooBar resources
type FooBarList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []FooBar `json:"items"`
}

// Test that a FooBar can be assigned as a GenericCRD.
var _ webhook.GenericCRD = &FooBar{}

func (r *FooBar) GetObjectMeta() metav1.Object {
	return &r.ObjectMeta
}

func (r *FooBar) GetSpecJSON() ([]byte, error) {
	return json.Marshal(r.Spec)
}
