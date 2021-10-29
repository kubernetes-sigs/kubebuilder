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
/* */
package owned_resource

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)
// +kubebuilder:docs-gen:collapse=Imports

/*
In this example the controller is doing basic management of a Deployment object.

The Spec here allows the user to customize the deployment created in various ways.
For example, the number of replicas it runs with.
*/

// SimpleDeploymentSpec defines the desired state of SimpleDeployment
type SimpleDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The number of replicas that the deployment should have
	// +optional
	Replicas *int `json:"replicas,omitempty"`
}

/*
The rest of the API configuration is covered in the CronJob tutorial.
*/
/* */
// SimpleDeploymentStatus defines the observed state of SimpleDeployment
type SimpleDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SimpleDeployment is the Schema for the simpledeployments API
type SimpleDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SimpleDeploymentSpec   `json:"spec,omitempty"`
	Status SimpleDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SimpleDeploymentList contains a list of SimpleDeployment
type SimpleDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SimpleDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SimpleDeployment{}, &SimpleDeploymentList{})
}
// +kubebuilder:docs-gen:collapse=Remaining API Code