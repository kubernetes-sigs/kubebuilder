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
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &ControllerTest{}

// ControllerTest scaffolds the file that sets up the controller unit tests
//

type ControllerTest struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	Force bool

	DoAPI bool
}

// SetTemplateDefaults implements machinery.Template
func (f *ControllerTest) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join("internal", "controller", "%[group]", "%[kind]_controller_test.go")
		} else {
			f.Path = filepath.Join("internal", "controller", "%[kind]_controller_test.go")
		}
	}

	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Println(f.Path)

	f.TemplateBody = controllerTestTemplate

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	}

	return nil
}

const controllerTestTemplate = `{{ .Boilerplate }}

{{if and .MultiGroup .Resource.Group }}
package {{ .Resource.PackageName }}
{{else}}
package controller
{{end}}

import (
	{{ if .DoAPI -}}
	"context"
	{{- end }}
	. "github.com/onsi/ginkgo/v2"
	{{ if .DoAPI -}}

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	{{ if not (isEmptyStr .Resource.Path) -}}
	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
	{{- end }}
	{{- end }}
)

var _ = Describe("{{ .Resource.Kind }} Controller", func() {
	Context("When reconciling a resource", func() {
		{{ if .DoAPI -}}
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",  // TODO(user):Modify as needed
		}
		{{ lower .Resource.Kind }} := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind {{ .Resource.Kind }}")
			err := k8sClient.Get(ctx, typeNamespacedName, {{ lower .Resource.Kind }})
			if err != nil && errors.IsNotFound(err) {
				resource := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance {{ .Resource.Kind }}")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		{{- end }}
		It("should successfully reconcile the resource", func() {
			{{ if .DoAPI -}}
			By("Reconciling the created resource")
			controllerReconciler := &{{ .Resource.Kind }}Reconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			{{- end }}
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
`
