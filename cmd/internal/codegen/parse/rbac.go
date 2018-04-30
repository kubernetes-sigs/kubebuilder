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

package parse

import (
	"fmt"
	"log"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/gengo/types"
)

// parseRBAC populates the RBAC rules for each annotated type.
func (b *APIs) parseRBAC() {
	for _, c := range b.context.Order {
		if IsRBAC(c) {
			for _, tag := range b.getRBACTag(c) {
				b.Rules = append(b.Rules, parseRBACTag(tag))
			}
		}
	}
}

func (b *APIs) getRBACTag(c *types.Type) []string {
	comments := Comments(c.CommentLines)
	resource := comments.getTags("rbac", ":")
	resource = append(resource, comments.getTags("kubebuilder:rbac", ":")...)
	if len(resource) == 0 {
		panic(fmt.Errorf("Must specify +kubebuilder:rbac comment for type %v", c.Name))
	}
	return resource
}

func parseRBACTag(tag string) rbacv1.PolicyRule {
	result := rbacv1.PolicyRule{}
	for _, elem := range strings.Split(tag, ",") {
		kv := strings.Split(elem, "=")
		if len(kv) != 2 {
			log.Fatalf("// +kubebuilder:rbac: tags must be key value pairs.  Expected "+
				"keys [groups=<group1;group2>,resources=<resource1;resource2>,verbs=<verb1;verb2>] "+
				"Got string: [%s]", tag)
		}
		value := kv[1]
		values := []string{}
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = value[1 : len(value)-1]
		}
		values = strings.Split(value, ";")
		switch kv[0] {
		case "groups":
			result.APIGroups = values
		case "resources":
			result.Resources = values
		case "verbs":
			result.Verbs = values
		case "urls":
			result.NonResourceURLs = values
		}
	}
	return result
}

// parseInfomers populates the informers to generate on each annotated type.
func (b *APIs) parseInformers() {
	for _, c := range b.context.Order {
		if IsInformer(c) {
			for _, tag := range b.getInformerTag(c) {
				if b.Informers == nil {
					b.Informers = map[v1.GroupVersionKind]bool{}
				}
				b.Informers[parseInformerTag(tag)] = true
			}
		}
	}
}

func (b *APIs) getInformerTag(c *types.Type) []string {
	comments := Comments(c.CommentLines)
	resource := comments.getTags("informers", ":")
	resource = append(resource, comments.getTags("kubebuilder:informers", ":")...)
	if len(resource) == 0 {
		panic(fmt.Errorf("Must specify +kubebuilder:informers comment for type %v", c.Name))
	}
	return resource
}

func parseInformerTag(tag string) v1.GroupVersionKind {
	result := v1.GroupVersionKind{}
	for _, elem := range strings.Split(tag, ",") {
		kv := strings.Split(elem, "=")
		if len(kv) != 2 {
			log.Fatalf("// +kubebuilder:informers: tags must be key value pairs.  Expected "+
				"keys [group=core,version=v1,kind=Pod] "+
				"Got string: [%s]", tag)
		}
		value := kv[1]
		switch kv[0] {
		case "group":
			result.Group = value
		case "version":
			result.Version = value
		case "kind":
			result.Kind = value
		}
	}
	return result
}
