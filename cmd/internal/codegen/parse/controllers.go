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
	"log"
	"strings"

	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/gengo/types"
)

// resourceTags contains the tags present in a "+resource=" comment
type controllerTags struct {
	gvk      schema.GroupVersionKind
	resource string
}

// parseControllers populates the list of controllers to generate code from the
// list of annotated types.
func (b *APIs) parseControllers() {
	for _, c := range b.context.Order {
		if IsController(c) {
			tags := parseControllerTag(b.getControllerTag(c))
			repo := strings.Split(c.Name.Package, "/pkg/controller")[0]
			pkg := b.context.Universe[c.Name.Package]
			b.Controllers = append(b.Controllers, codegen.Controller{
				tags.gvk, tags.resource, pkg, repo})
		}
	}
}

func (b *APIs) getControllerTag(c *types.Type) string {
	comments := Comments(c.CommentLines)
	resource := comments.getTag("controller", ":") + comments.getTag("kubebuilder:controller", ":")
	if len(resource) == 0 {
		panic(errors.Errorf("Must specify +kubebuilder:controller comment for type %v", c.Name))
	}
	return resource
}

// parseResourceTag parses the tags in a "+resource=" comment into a resourceTags struct
func parseControllerTag(tag string) controllerTags {
	result := controllerTags{}
	for _, elem := range strings.Split(tag, ",") {
		kv := strings.Split(elem, "=")
		if len(kv) != 2 {
			log.Fatalf("// +kubebuilder:controller: tags must be key value pairs.  Expected "+
				"keys [group=<group>,version=<version>,kind=<kind>,resource=<resource>] "+
				"Got string: [%s]", tag)
		}
		value := kv[1]
		switch kv[0] {
		case "group":
			result.gvk.Group = value
		case "version":
			result.gvk.Version = value
		case "kind":
			result.gvk.Kind = value
		case "resource":
			result.resource = value
		}
	}
	return result
}
