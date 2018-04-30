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

	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen"
	"github.com/pkg/errors"
	"k8s.io/gengo/types"
)

// parseIndex indexes all types with the comment "// +resource=RESOURCE" by GroupVersionKind and
// GroupKindVersion
func (b *APIs) parseIndex() {
	// Index resource by group, version, kind
	b.ByGroupVersionKind = map[string]map[string]map[string]*codegen.APIResource{}

	// Index resources by group, kind, version
	b.ByGroupKindVersion = map[string]map[string]map[string]*codegen.APIResource{}

	// Index subresources by group, version, kind
	b.SubByGroupVersionKind = map[string]map[string]map[string]*types.Type{}

	for _, c := range b.context.Order {
		// The type is a subresource, add it to the subresource index
		if IsAPISubresource(c) {
			group := GetGroup(c)
			version := GetVersion(c, group)
			kind := GetKind(c, group)
			if _, f := b.SubByGroupVersionKind[group]; !f {
				b.SubByGroupVersionKind[group] = map[string]map[string]*types.Type{}
			}
			if _, f := b.SubByGroupVersionKind[group][version]; !f {
				b.SubByGroupVersionKind[group][version] = map[string]*types.Type{}
			}
			b.SubByGroupVersionKind[group][version][kind] = c
		}

		// If it isn't a subresource or resource, continue to the next type
		if !IsAPIResource(c) {
			continue
		}

		// Parse out the resource information
		r := &codegen.APIResource{
			Type:          c,
			NonNamespaced: IsNonNamespaced(c),
		}
		r.Group = GetGroup(c)
		r.Version = GetVersion(c, r.Group)
		r.Kind = GetKind(c, r.Group)
		r.Domain = b.Domain

		rt := parseResourceTag(b.getResourceTag(c))
		r.Resource = rt.Resource
		r.ShortName = rt.ShortName
		//r.REST = rt.REST
		//r.Strategy = rt.Strategy

		// Copy the Status strategy to mirror the non-status strategy
		r.StatusStrategy = strings.TrimSuffix(r.Strategy, "Strategy")
		r.StatusStrategy = fmt.Sprintf("%sStatusStrategy", r.StatusStrategy)

		// Initialize the map entries so they aren't nill
		if _, f := b.ByGroupKindVersion[r.Group]; !f {
			b.ByGroupKindVersion[r.Group] = map[string]map[string]*codegen.APIResource{}
		}
		if _, f := b.ByGroupKindVersion[r.Group][r.Kind]; !f {
			b.ByGroupKindVersion[r.Group][r.Kind] = map[string]*codegen.APIResource{}
		}
		if _, f := b.ByGroupVersionKind[r.Group]; !f {
			b.ByGroupVersionKind[r.Group] = map[string]map[string]*codegen.APIResource{}
		}
		if _, f := b.ByGroupVersionKind[r.Group][r.Version]; !f {
			b.ByGroupVersionKind[r.Group][r.Version] = map[string]*codegen.APIResource{}
		}

		// Add the resource to the map
		b.ByGroupKindVersion[r.Group][r.Kind][r.Version] = r
		b.ByGroupVersionKind[r.Group][r.Version][r.Kind] = r

		//if !HasSubresource(c) {
		//	continue
		//}
		r.Type = c
		//r.Subresources = b.getSubresources(r)
	}
}

//func (b *APIs) getSubresources(c *codegen.APIResource) map[string]*codegen.APISubresource {
//	r := map[string]*codegen.APISubresource{}
//	subresources := b.getSubresourceTags(c.Type)
//
//	if len(subresources) == 0 {
//		// Not a subresource
//		return r
//	}
//for _, subresource := range subresources {
//	// Parse the values for each subresource
//	tags := parseSubresourceTag(c, subresource)
//	sr := &codegen.APISubresource{
//		Kind:     tags.Kind,
//		Request:  tags.RequestKind,
//		Path:     tags.Path,
//		REST:     tags.REST,
//		Domain:   b.Domain,
//		Version:  c.Version,
//		Resource: c.Resource,
//		Group:    c.Group,
//	}
//	if !b.isInPackage(tags) {
//		// Out of package Request types require an import and are prefixed with the
//		// package name - e.g. v1.Scale
//		sr.Request, sr.ImportPackage = b.getNameAndImport(tags)
//	}
//	if v, found := r[sr.Path]; found {
//		log.Fatalf("Multiple subresources registered for path %s: %v %v",
//			sr.Path, v, subresource)
//	}
//	r[sr.Path] = sr
//}
//	return r
//}

// subresourceTags contains the tags present in a "+subresource=" comment
//type subresourceTags struct {
//	Path        string
//	Kind        string
//	RequestKind string
//	REST        string
//}
//
//func (b *APIs) getSubresourceTags(c *types.Type) []string {
//	comments := Comments(c.CommentLines)
//	return comments.getTags("subresource", ":")
//}

// Returns true if the subresource Request type is in the same package as the resource type
//func (b *APIs) isInPackage(tags subresourceTags) bool {
//	return !strings.Contains(tags.RequestKind, ".")
//}
//
//// GetNameAndImport converts
//func (b *APIs) getNameAndImport(tags subresourceTags) (string, string) {
//	last := strings.LastIndex(tags.RequestKind, ".")
//	importPackage := tags.RequestKind[:last]
//
//	// Set the request kind to the struct name
//	tags.RequestKind = tags.RequestKind[last+1:]
//	// Find the package
//	pkg := filepath.Base(importPackage)
//	// Prefix the struct name with the package it is in
//	return strings.Join([]string{pkg, tags.RequestKind}, "."), importPackage
//}

// resourceTags contains the tags present in a "+resource=" comment
type resourceTags struct {
	Resource  string
	REST      string
	Strategy  string
	ShortName string
}

// ParseResourceTag parses the tags in a "+resource=" comment into a resourceTags struct
func parseResourceTag(tag string) resourceTags {
	result := resourceTags{}
	for _, elem := range strings.Split(tag, ",") {
		kv := strings.Split(elem, "=")
		if len(kv) != 2 {
			log.Fatalf("// +kubebuilder:resource: tags must be key value pairs.  Expected "+
				"keys [path=<subresourcepath>] "+
				"Got string: [%s]", tag)
		}
		value := kv[1]
		switch kv[0] {
		//case "rest":
		//	result.REST = value
		case "path":
			result.Resource = value
		//case "strategy":
		//	result.Strategy = value
		case "shortName":
			result.ShortName = value
		}
	}
	return result
}

// ParseSubresourceTag parses the tags in a "+subresource=" comment into a subresourceTags struct
//func parseSubresourceTag(c *codegen.APIResource, tag string) subresourceTags {
//	result := subresourceTags{}
//	for _, elem := range strings.Split(tag, ",") {
//		kv := strings.Split(elem, "=")
//		if len(kv) != 2 {
//			log.Fatalf("// +subresource: tags must be key value pairs.  Expected "+
//				"keys [request=<requestType>,rest=<restImplType>,path=<subresourcepath>] "+
//				"Got string: [%s]", tag)
//		}
//		value := kv[1]
//		switch kv[0] {
//		case "request":
//			result.RequestKind = value
//		case "rest":
//			result.REST = value
//		case "path":
//			// Strip the parent resource
//			result.Path = strings.Replace(value, c.Resource+"/", "", -1)
//		}
//	}
//	return result
//}

// getResourceTag returns the value of the "+resource=" comment tag
func (b *APIs) getResourceTag(c *types.Type) string {
	comments := Comments(c.CommentLines)
	resource := comments.getTag("resource", ":") + comments.getTag("kubebuilder:resource", ":")
	if len(resource) == 0 {
		panic(errors.Errorf("Must specify +kubebuilder:resource comment for type %v", c.Name))
	}
	return resource
}
