/*
Copyright 2023 The Kubernetes authors.

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

package v2

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// See https://book.kubebuilder.io/multiversion-tutorial/conversion.html.
func (src *FirstMate) ConvertTo(dstRaw conversion.Hub) error {
	// Implement your logic here to convert from hub to spoke version.
	return nil
}
func (dst *FirstMate) ConvertFrom(srcRaw conversion.Hub) error {
	// Implement your logic here to convert from spoke to hub version.
	return nil
}
