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
	"math"
	"sort"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var _ kio.Filter = &SortFilter{}

// SortFilter sorts resources so they are installed in the right order
type SortFilter struct {
	*KubebuilderConfigGen
}

var order = func() map[string]int {
	m := map[string]int{}
	for i, k := range []string{
		"Namespace", "CustomResourceDefinition", "Role", "ClusterRole",
		"RoleBinding", "ClusterRoleBinding", "Service", "Secret", "Deployment",
	} {
		m[k] = i + 1
	}
	return m
}()

// Filter implements kio.Filter
func (cgr SortFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	sort.Slice(input, func(i, j int) bool {
		mi, _ := input[i].GetMeta()
		mj, _ := input[j].GetMeta()
		oi := order[mi.Kind]
		if oi == 0 {
			oi = math.MaxInt32
		}
		oj := order[mj.Kind]
		if oj == 0 {
			oj = math.MaxInt32
		}
		if oi != oj {
			return oi < oj
		}
		return strings.Compare(mi.Name, mj.Name) < 0
	})
	return input, nil
}
