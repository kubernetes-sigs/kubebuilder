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
	"path/filepath"

	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen"
	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/gengo/args"
	"k8s.io/gengo/generator"
	"k8s.io/gengo/types"
)

type APIs struct {
	context         *generator.Context
	arguments       *args.GeneratorArgs
	Domain          string
	VersionedPkgs   sets.String
	UnversionedPkgs sets.String
	APIsPkg         string
	APIsPkgRaw      *types.Package
	GroupNames      sets.String

	APIs        *codegen.APIs
	Controllers []codegen.Controller

	ByGroupKindVersion    map[string]map[string]map[string]*codegen.APIResource
	ByGroupVersionKind    map[string]map[string]map[string]*codegen.APIResource
	SubByGroupVersionKind map[string]map[string]map[string]*types.Type
	Groups                map[string]types.Package
	Rules                 []rbacv1.PolicyRule
}

func NewAPIs(context *generator.Context, arguments *args.GeneratorArgs) *APIs {
	b := &APIs{
		context:   context,
		arguments: arguments,
	}
	b.parsePackages()
	b.parseDomain()
	b.parseGroupNames()
	b.parseIndex()
	b.parseControllers()
	b.parseRBAC()
	b.parseAPIs()
	b.parseJSONSchemaProps()
	return b
}

// parseGroupNames initializes b.GroupNames with the set of all groups
func (b *APIs) parseGroupNames() {
	b.GroupNames = sets.String{}
	for p := range b.UnversionedPkgs {
		pkg := b.context.Universe[p]
		if pkg == nil {
			// If the input had no Go files, for example.
			continue
		}
		b.GroupNames.Insert(filepath.Base(p))
	}
}

// parsePackages parses out the sets of Versioned, Unversioned packages and identifies the root Apis package.
func (b *APIs) parsePackages() {
	b.VersionedPkgs = sets.NewString()
	b.UnversionedPkgs = sets.NewString()
	for _, o := range b.context.Order {
		if IsAPIResource(o) {
			versioned := o.Name.Package
			b.VersionedPkgs.Insert(versioned)

			unversioned := filepath.Dir(versioned)
			b.UnversionedPkgs.Insert(unversioned)

			if apis := filepath.Dir(unversioned); apis != b.APIsPkg && len(b.APIsPkg) > 0 {
				panic(errors.Errorf(
					"Found multiple apis directory paths: %v and %v.  "+
						"Do you have a +resource tag on a resource that is not in a version "+
						"directory?", b.APIsPkg, apis))
			} else {
				b.APIsPkg = apis
			}
		}
	}
}

// parseDomain parses the domain from the apis/doc.go file comment "// +domain=YOUR_DOMAIN".
func (b *APIs) parseDomain() {
	pkg := b.context.Universe[b.APIsPkg]
	if pkg == nil {
		// If the input had no Go files, for example.
		panic(errors.Errorf("Missing apis package."))
	}
	comments := Comments(pkg.Comments)
	b.Domain = comments.getTag("domain", "=")
	if len(b.Domain) == 0 {
		panic("Could not find string matching // +domain=.+ in apis/doc.go")
	}
}
