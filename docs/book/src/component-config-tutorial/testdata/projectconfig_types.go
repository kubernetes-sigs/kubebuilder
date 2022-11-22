/*
Copyright 2020 The Kubernetes authors.

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
We start out simply enough: we import the `config/v1alpha1` API group, which is
exposed through ControllerRuntime.
*/
package v2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
)

// +kubebuilder:object:root=true

/*
Next, we'll remove the default `ProjectConfigSpec` and `ProjectConfigList` then
we'll embed `cfg.ControllerManagerConfigurationSpec` in `ProjectConfig`.
*/

// ProjectConfig is the Schema for the projectconfigs API
type ProjectConfig struct {
	metav1.TypeMeta `json:",inline"`

	// ControllerManagerConfigurationSpec returns the configurations for controllers
	cfg.ControllerManagerConfigurationSpec `json:",inline"`

	ClusterName string `json:"clusterName,omitempty"`
}

/*
If you haven't, you'll also need to remove the `ProjectConfigList` from the
`SchemeBuilder.Register`.
*/
func init() {
	SchemeBuilder.Register(&ProjectConfig{})
}
