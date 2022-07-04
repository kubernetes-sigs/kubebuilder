/*
Copyright 2022 The Kubernetes Authors.

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

package controllers

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &ControllerTest{}

// ControllerTest scaffolds the file that defines tests for the controller for a CRD or a builtin resource
// nolint:maligned
type ControllerTest struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	Image string
}

// SetTemplateDefaults implements file.Template
func (f *ControllerTest) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join("controllers", "%[group]", "%[kind]_controller_test.go")
		} else {
			f.Path = filepath.Join("controllers", "%[kind]_controller_test.go")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	fmt.Println("creating import for %", f.Resource.Path)
	f.TemplateBody = controllerTestTemplate

	return nil
}

//nolint:lll
const controllerTestTemplate = `{{ .Boilerplate }}

package {{ if and .MultiGroup .Resource.Group }}{{ .Resource.PackageName }}{{ else }}controllers{{ end }}

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"time"
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	
	{{ if not (isEmptyStr .Resource.Path) -}}
	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
	{{- end }}
)

var _ = Describe("{{ .Resource.Kind }} controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		{{ .Resource.Kind }}Name      = "test-{{ lower .Resource.Kind }}"
		{{ .Resource.Kind }}Namespace = "test-{{ lower .Resource.Kind }}"
	)

	Context("{{ .Resource.Kind }} controller test", func() {
		It("should create successfully the custom resource for the {{ .Resource.Kind }}", func() {
			ctx := context.Background()

			By("Creating the custom resource for the Kind {{ .Resource.Kind }}")
			{{ lower .Resource.Kind }} := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: {{ .Resource.Kind }}Name, Namespace: {{ .Resource.Kind }}Namespace}, {{ lower .Resource.Kind }})
			if err != nil && errors.IsNotFound(err) {
				// Define a new custom resource
				{{ lower .Resource.Kind }} := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "{{ .Resource.Group }}.{{ .Resource.Domain }}/{{ .Resource.Version }}",
						Kind:       "{{ .Resource.Kind }}",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      {{ .Resource.Kind }}Name,
						Namespace: {{ .Resource.Kind }}Namespace,
					},
					Spec: {{ .Resource.ImportAlias }}.{{ .Resource.Kind }}Spec{
						Size: 1,
					},
				}
				fmt.Fprintf(GinkgoWriter, fmt.Sprintf("Creating a new custom resource in the namespace: %s with the name %s\n", {{ lower .Resource.Kind }}.Namespace, {{ lower .Resource.Kind }}.Name))
				err = k8sClient.Create(ctx, {{ lower .Resource.Kind }})
				if err != nil {
					Expect(err).To(Not(HaveOccurred()))
				}
			} 

			By("Checking with {{ .Resource.Kind }} Kind exist")
			Eventually(func() error {
				found := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
				err = k8sClient.Get(ctx, types.NamespacedName{Name: {{ .Resource.Kind }}Name, Namespace: {{ .Resource.Kind }}Namespace}, found)
				if err != nil {
					return err
				}
				return nil
			}, time.Minute, time.Second).Should(Succeed())
		})
	})

})
`
