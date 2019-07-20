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

package v3

import (
	"fmt"
	"strconv"
	"strings"

	diskapiv1 "infra.kubebuilder.io/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:docs-gen:collapse=Imports

/*
In v2 iteration of our API, we evolved price field to a better structure. In v3 iteration
we decided to rename `price` field to `PricePerGB` to make it more explicit since
the price field represents price/per/GB.
*/

// DiskSpec defines the desired state of Disk
type DiskSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster

	// PricePerGB represents price for the disk.
	PricePerGB Price `json:"pricePerGB"`
}

// Price represents a generic price value that has amount and currency.
type Price struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

/*
 */

// DiskStatus defines the observed state of Disk
type DiskStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// Disk is the Schema for the disks API
type Disk struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DiskSpec   `json:"spec,omitempty"`
	Status DiskStatus `json:"status,omitempty"`
}

// +kubebuilder:docs-gen:collapse=Type definitions

/*
The v3 version has two changes in price field
- New structure of price field (same as v2 version)
- Renaming of the field to `PricePerGB`

Since v3 is a spoke version, v3 type is required to implement
`conversion.Convertible` interface. Now, let's take a look at the conversion methods.
*/

// ConvertTo converts receiver (v3.Disk instance in this case) to provided Hub
// instance (v1.Disk in our case).
func (disk *Disk) ConvertTo(dst conversion.Hub) error {
	switch t := dst.(type) {
	case *diskapiv1.Disk:
		diskv1 := dst.(*diskapiv1.Disk)
		diskv1.ObjectMeta = disk.ObjectMeta
		// conversion implementation
		// in our case, we convert the price in structured form to string form.
		// Note that we use the value from PricePerGB field.
		diskv1.Spec.Price = fmt.Sprintf("%d %s",
			disk.Spec.PricePerGB.Amount, disk.Spec.PricePerGB.Currency)
		return nil
	default:
		return fmt.Errorf("unsupported type %v", t)
	}
}

// ConvertFrom converts provided Hub instance (v1.Disk in our case) to receiver
// (v3.Disk in our case).
func (disk *Disk) ConvertFrom(src conversion.Hub) error {
	switch t := src.(type) {
	case *diskapiv1.Disk:
		diskv1 := src.(*diskapiv1.Disk)
		disk.ObjectMeta = diskv1.ObjectMeta
		// conversion implementation
		// Note that the conversion logic is same as we implement for v2 except
		// that we use PricePerGB instead of Price.
		parts := strings.Fields(diskv1.Spec.Price)
		if len(parts) != 2 {
			return fmt.Errorf("invalid price")
		}
		amount, err := strconv.Atoi(parts[0])
		if err != nil {
			return err
		}
		disk.Spec.PricePerGB = Price{
			Amount:   int64(amount),
			Currency: parts[1],
		}
		return nil
	default:
		return fmt.Errorf("unsupported type %v", t)
	}
}

/*
 */

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

// +kubebuilder:docs-gen:collapse=List definition
