/*
Copyright 2019 The Kubernetes Authors.

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
package v2

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/resource"
)

var _ input.File = &TypesTest{}

// TypesTest scaffolds the api/version/kind_types_test.go file to test the API schema
type TypesTest struct {
	input.Input

	// Resource is the resource to scaffold the types_test.go file for
	Resource *resource.Resource
}

// GetInput implements input.File
func (t *TypesTest) GetInput() (input.Input, error) {
	if t.Path == "" {
		t.Path = filepath.Join("api", t.Resource.Version,
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

var typesTestTemplate = `{{ .Boilerplate }}

package {{ .Resource.Version }}

import (
	"testing"
	
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// These tests are written in BDD-style using Ginkgo framework. Refer to 
// http://onsi.github.io/ginkgo to learn more.


var _ = Describe("{{ .Resource.Kind }}", func() {
	var (
		key types.NamespacedName
		created, fetched *{{.Resource.Kind}}
	)

	BeforeEach(func(){
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func(){
		// Add any teardown steps that needs to be executed after each test
	})
 
	// Add Tests for OpenAPI validation (or additonal CRD features) specified in
	// your API definition.
	// Avoid adding tests for vanilla CRUD operations because they would
	// test Kubernetes API server, which isn't the goal here.
	Context("Create API", func() {

		It("should create an object successfully", func() {

		 key = types.NamespacedName{
				 Name: "foo",
				 {{ if .Resource.Namespaced -}}
				 Namespace: "default",
				 {{ end -}}
		 }
		 created = &{{ .Resource.Kind }}{
				 ObjectMeta: metav1.ObjectMeta{
						 Name: "foo",
						 {{ if .Resource.Namespaced -}}
						 Namespace: "default",
						 {{ end -}}
				 }}
		
		 By("creating an API obj")
		 Expect(k8sClient.Create(context.TODO(), created)).To(Succeed())

		 fetched = &{{ .Resource.Kind }}{}
		 Expect(k8sClient.Get(context.TODO(), key, fetched)).To(Succeed())
		 Expect(fetched).To(Equal(created))

		 By("deleting the created object")
		 Expect(k8sClient.Delete(context.TODO(), created)).To(Succeed())
		 Expect(k8sClient.Get(context.TODO(), key, created)).ToNot(Succeed())
		})

	})

})
`
