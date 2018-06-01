package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!
// Created by "kubebuilder create resource" for you to implement the TestKind resource schema definition
// as a go struct.
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TestKindSpec defines the desired state of TestKind
type TestKindSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "kubebuilder generate" to regenerate code after modifying this file
	Count int `json:"count"`

	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:ExclusiveMinimum=true
	Power  float32 `json:"power,omitempty"`
	Bricks int32   `json:"bricks,omitempty"`
	// +kubebuilder:validation:MaxLength=15
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:MaxItems=500
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:UniqueItems=false
	Knights []string `json:"knights,omitempty"`
	Winner  bool     `json:"winner,omitempty"`
	// +kubebuilder:validation:Enum=Lion,Wolf,Dragon
	Alias string `json:"alias,omitempty"`
	// +kubebuilder:validation:Enum=1,2,3
	Rank    int    `json:"rank"`
	Comment []byte `json:"comment,omitempty"`
}

// TestKindStatus defines the observed state of TestKind
type TestKindStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "kubebuilder generate" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TestKind
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=testkinds
type TestKind struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestKindSpec   `json:"spec,omitempty"`
	Status TestKindStatus `json:"status,omitempty"`
}
