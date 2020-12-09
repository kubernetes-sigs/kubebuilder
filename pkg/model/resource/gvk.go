/*
Copyright 2020 The Kubernetes Authors.

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

package resource

// GVK holds the unique identifier of a resource
type GVK struct {
	// Group is the qualified resource's group.
	Group string

	// Version is the resource's version.
	Version string

	// Kind is the resource's kind.
	Kind string
}

// IsEqualTo compares two GVK objects.
func (a GVK) IsEqualTo(b GVK) bool {
	return a.Group == b.Group &&
		a.Version == b.Version &&
		a.Kind == b.Kind
}
