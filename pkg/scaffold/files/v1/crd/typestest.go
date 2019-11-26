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

package crd

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
)

var _ input.File = &TypesTest{}

// TypesTest scaffolds the pkg/apis/group/version/kind_types_test.go file to test the API schema
type TypesTest struct {
	input.Input

	// Resource is the resource to scaffold the types_test.go file for
	Resource *resource.Resource
}

// GetInput implements input.File
func (t *TypesTest) GetInput() (input.Input, error) {
	if t.Path == "" {
		t.Path = filepath.Join("pkg", "apis", t.Resource.Group, t.Resource.Version,
			fmt.Sprintf("%s_types_test.go", strings.ToLower(t.Resource.Kind)))
	}
	t.TemplateBody = typesTestTemplate
	t.IfExistsAction = input.Error
	return t.Input, nil
}

// Validate validates the values
func (t *TypesTest) Validate() error {
	return t.Resource.Validate()
}

const typesTestTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	"testing"

	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestStorage{{ .Resource.Kind }}(t *testing.T) {
     key := types.NamespacedName{
             Name: "foo",
             {{ if .Resource.Namespaced -}}
             Namespace: "default",
             {{ end -}}
     }
     created := &{{ .Resource.Kind }}{
             ObjectMeta: metav1.ObjectMeta{
                     Name: "foo",
                     {{ if .Resource.Namespaced -}}
                     Namespace: "default",
                     {{ end -}}
             }}
	g := gomega.NewGomegaWithT(t)

	// Test Create
	fetched := &{{ .Resource.Kind }}{}
	g.Expect(c.Create(context.TODO(), created)).To(gomega.Succeed())

	g.Expect(c.Get(context.TODO(), key, fetched)).To(gomega.Succeed())
	g.Expect(fetched).To(gomega.Equal(created))

	// Test Updating the Labels
	updated := fetched.DeepCopy()
	updated.Labels = map[string]string{"hello": "world"}
	g.Expect(c.Update(context.TODO(), updated)).To(gomega.Succeed())

	g.Expect(c.Get(context.TODO(), key, fetched)).To(gomega.Succeed())
	g.Expect(fetched).To(gomega.Equal(updated))

	// Test Delete
	g.Expect(c.Delete(context.TODO(), fetched)).To(gomega.Succeed())
	g.Expect(c.Get(context.TODO(), key, fetched)).ToNot(gomega.Succeed())
}
`
