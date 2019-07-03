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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:docs-gen:collapse=Imports
// First version of our Spec has one field price of type string. We will evolve
// this field over the next two versions.

// DiskSpec defines the desired state of Disk
type DiskSpec struct {
	// Price represents price per GB for a Disk. It is specified in the
	// the format "<AMOUNT> <CURRENCY>". Example values will be "10 USD", "100 USD"
	Price string `json:"price"`
}

/*
We need to specify the version that is being used as storage version. In this
case, we decided to use v1 version for storage, so we use crd marker `+kubebuilder:storageversion`
on v1.Disk type to indicate that.
*/

// +kubebuilder:object:root=true

// +kubebuilder:storageversion
// Disk is the Schema for the disks API
type Disk struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DiskSpec   `json:"spec,omitempty"`
	Status DiskStatus `json:"status,omitempty"`
}

/*
We need to define a Hub type to faciliate conversion. Storage and hub version don't have to be same, but to keep things simple, we will specify v1 to be the Hub for the Disk kind. A version needs to implement [conversion.Hub interface](https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/conversion) to indicate that it is a Hub type. Given that v1 is a Hub version, it doesn't need to implement any conversion functions.
*/

/*
Next we define [Hub() method](https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/conversion) to indicate that v1 is the hub type
*/
// implements conversion.Hub interface.
func (disk *Disk) Hub() {}

// DiskStatus defines the observed state of Disk
type DiskStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// DiskList contains a list of Disk
type DiskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Disk `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Disk{}, &DiskList{})
}
