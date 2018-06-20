/*
Copyright 2017 The Kubernetes Authors.

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

/*
The apis package describes the comment directives that may be applied to apis / resources
*/
package apis

const (
	// Resource annotates a type as a resource
	Resource = "// +kubebuilder:resource:path="

	// StatusSubresource annotates a type as having a status subresource
	StatusSubresource = "// +kubebuilder:subresource:status"

	// Categories annotates a type as belonging to a comma-delimited list of
	// categories
	Categories = "// +kubebuilder:categories="

	// Maximum annotates a numeric go struct field for CRD validation
	Maximum = "// +kubebuilder:validation:Maximum="

	// ExclusiveMaximum annotates a numeric go struct field for CRD validation
	ExclusiveMaximum = "// +kubebuilder:validation:ExclusiveMaximum="

	// Minimum annotates a numeric go struct field for CRD validation
	Minimum = "// +kubebuilder:validation:Minimum="

	// ExclusiveMinimum annotates a numeric go struct field for CRD validation
	ExclusiveMinimum = "// +kubebuilder:validation:ExclusiveMinimum="

	// Pattern annotates a string go struct field for CRD validation with a regular expression it must match
	Pattern = "// +kubebuilder:validation:Pattern="

	// Enum specifies the valid values for a field
	Enum = "// +kubebuilder:validation:Enum="

	// MaxLength specifies the maximum length of a string field
	MaxLength = "// +kubebuilder:validation:MaxLength="

	// MinLength specifies the minimum length of a string field
	MinLength = "// +kubebuilder:validation:MinLength="

	// MaxItems specifies the maximum number of items an array or slice field may contain
	MaxItems = "// +kubebuilder:validation:MaxItems="

	// MinItems specifies the minimum number of items an array or slice field may contain
	MinItems = "// +kubebuilder:validation:MinItems="

	// UniqueItems specifies that all values in an array or slice must be unique
	UniqueItems = "// +kubebuilder:validation:UniqueItems="

	// Format annotates a string go struct field for CRD validation with a specific format
	Format = "// +kubebuilder:validation:Format="
)
