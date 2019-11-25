/*

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
// +kubebuilder:docs-gen:collapse=Apache License

/*
We start out simply enough: we import the `meta/v1` API group, which is not
normally exposed by itself, but instead contains metadata common to all
Kubernetes Kinds.
*/
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*
Next, we define types for the Spec and Status of our Kind.  Kubernetes functions
by reconciling desired state (`Spec`) with actual cluster state (other objects'
`Status`) and external state, and then recording what it observed (`Status`).
Thus, every *functional* object includes spec and status.  A few types, like
`ConfigMap` don't follow this pattern, since they don't encode desired state,
but most types do.
*/
// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CronJobSpec defines the desired state of CronJob
type CronJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// CronJobStatus defines the observed state of CronJob
type CronJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

/*
Next, we define the types corresponding to actual Kinds, `CronJob` and `CronJobList`.
`CronJob` is our root type, and describes the `CronJob` kind.  Like all Kubernetes objects, it contains
`TypeMeta` (which describes API version and Kind), and also contains `ObjectMeta`, which holds things
like name, namespace, and labels.

`CronJobList` is simply a container for multiple `CronJob`s.  It's the Kind used in bulk operations,
like LIST.

In general, we never modify either of these -- all modifications go in either Spec or Status.

That little `+kubebuilder:object:root` comment is called a marker.  We'll see
more of them in a bit, but know that they act as extra metadata, telling
[controller-tools](https://github.com/kubernetes-sigs/controller-tools) (our code and YAML generator) extra information.
This particular one tells the `object` generator that this type represents
a Kind.  Then, the `object` generator generates an implementation of the
[runtime.Object](https://godoc.org/k8s.io/apimachinery/pkg/runtime#Object) interface for us, which is the standard
interface that all types representing Kinds must implement.
*/

// +kubebuilder:object:root=true

// CronJob is the Schema for the cronjobs API
type CronJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CronJobSpec   `json:"spec,omitempty"`
	Status CronJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CronJobList contains a list of CronJob
type CronJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CronJob `json:"items"`
}

/*
Finally, we add the Go types to the API group.  This allows us to add the
types in this API group to any [Scheme](https://godoc.org/k8s.io/apimachinery/pkg/runtime#Scheme).
*/
func init() {
	SchemeBuilder.Register(&CronJob{}, &CronJobList{})
}
