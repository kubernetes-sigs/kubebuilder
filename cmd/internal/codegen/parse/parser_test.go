/*
Copyright 2018 The Kubernetes Authors.

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
package parse

import (
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestrbacMatchesInformers(t *testing.T) {
	tests := []struct {
		informers map[v1.GroupVersionKind]bool
		rbacRules []rbacv1.PolicyRule
		expErr    bool
	}{
		{
			// informer resource matches the RBAC rule
			informers: map[v1.GroupVersionKind]bool{
				v1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}: true,
			},
			rbacRules: []rbacv1.PolicyRule{
				{APIGroups: []string{"apps"}, Resources: []string{"deployments"}},
			},
			expErr: false,
		},
		{
			// RBAC rule does not match the informer resource because of missing pluralization in RBAC rules
			informers: map[v1.GroupVersionKind]bool{
				v1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}: true,
			},
			rbacRules: []rbacv1.PolicyRule{
				{APIGroups: []string{"apps"}, Resources: []string{"Deployment"}},
			},
			expErr: true,
		},
		{
			// wild-card RBAC rule should match any resource in the group
			informers: map[v1.GroupVersionKind]bool{
				v1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}: true,
				v1.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}:  true,
			},
			rbacRules: []rbacv1.PolicyRule{
				{APIGroups: []string{"apps"}, Resources: []string{"*"}},
			},
			expErr: false,
		},
		{
			// empty group name is normalized to "core"
			informers: map[v1.GroupVersionKind]bool{
				v1.GroupVersionKind{Group: "core", Version: "v1", Kind: "Pod"}: true,
			},
			rbacRules: []rbacv1.PolicyRule{
				{APIGroups: []string{""}, Resources: []string{"pods"}},
			},
			expErr: false,
		},
		{
			// empty group name is normalized to "core"
			informers: map[v1.GroupVersionKind]bool{
				v1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}: true,
			},
			rbacRules: []rbacv1.PolicyRule{
				{APIGroups: []string{"core"}, Resources: []string{"pods"}},
			},
			expErr: false,
		},
	}

	for _, test := range tests {
		err := checkRBACMatchesInformers(test.informers, test.rbacRules)
		if test.expErr {
			if err == nil {
				t.Errorf("RBAC rules %+v shouldn't match with informers %+v", test.rbacRules, test.informers)
			}
		} else {
			if err != nil {
				t.Errorf("RBAC rules %+v should match informers %+v, but got a mismatch error: %v", test.rbacRules, test.informers, err)
			}
		}
	}
}
