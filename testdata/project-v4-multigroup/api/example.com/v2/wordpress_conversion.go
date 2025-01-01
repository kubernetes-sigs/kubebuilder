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

package v2

import (
	"log"

	"sigs.k8s.io/controller-runtime/pkg/conversion"

	examplecomv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/example.com/v1"
)

// ConvertTo converts this Wordpress (v2) to the Hub version (v1).
func (src *Wordpress) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*examplecomv1.Wordpress)
	log.Printf("ConvertTo: Converting Wordpress from Spoke version v2 to Hub version v1;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(user): Implement conversion logic from v2 to v1
	return nil
}

// ConvertFrom converts the Hub version (v1) to this Wordpress (v2).
func (dst *Wordpress) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*examplecomv1.Wordpress)
	log.Printf("ConvertFrom: Converting Wordpress from Hub version v1 to Spoke version v2;"+
		"source: %s/%s, target: %s/%s", src.Namespace, src.Name, dst.Namespace, dst.Name)

	// TODO(user): Implement conversion logic from v1 to v2
	return nil
}
