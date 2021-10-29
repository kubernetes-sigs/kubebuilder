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
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var _ kio.Filter = &ControllerGenFilter{}

// ComponentFilter inserts the component config read from disk into the ConfigMap
type ComponentFilter struct {
	*KubebuilderConfigGen
}

// Filter sets the component config in the configmap
func (cf ComponentFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	if !cf.Spec.ControllerManager.ComponentConfig.Enable {
		return input, nil
	}
	s := &framework.Selector{
		APIVersions: []string{"v1"},
		Kinds:       []string{"ConfigMap"},
		Names:       []string{"manager-config"},
		Namespaces:  []string{cf.Namespace},
	}
	matches, err := s.Filter(input)
	if err != nil {
		return nil, err
	}
	for i := range matches {
		m := matches[i]
		value := yaml.NewStringRNode(cf.Status.ComponentConfigString)
		value.YNode().Style = yaml.LiteralStyle
		err := m.PipeE(
			yaml.Lookup("data", "controller_manager_config.yaml"),
			yaml.FieldSetter{OverrideStyle: true, Value: value})
		if err != nil {
			return nil, err
		}
	}
	return input, nil
}
