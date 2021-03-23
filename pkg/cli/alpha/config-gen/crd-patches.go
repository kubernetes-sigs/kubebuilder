/*
Copyright 2021 The Kubernetes Authors.

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

package configgen

import (
	"github.com/markbates/pkger"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// CRDPatchTemplate returns the PatchTemplate for crd
func CRDPatchTemplate(kp *KubebuilderConfigGen) framework.PT {
	return framework.PT{
		// keep casting -- required by pkger to find the directory
		Dir: pkger.Dir("/pkg/cli/alpha/config-gen/templates/patches/crd"),
		Selector: func() *framework.Selector {
			return &framework.Selector{
				Kinds: []string{"CustomResourceDefinition"},
				Filter: func(r *yaml.RNode) bool {
					m, _ := r.GetMeta()
					return kp.Spec.Webhooks.Conversions[m.Name]
				},
			}
		},
	}
}
