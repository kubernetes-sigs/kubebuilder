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

// ControllerTest scaffolds the file that defines tests for the controller for a CRD or a builtin resource
//
//nolint:maligned
type ControllerTest struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	Port        string
	PackageName string
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

	f.PackageName = "controller"
	f.IfExistsAction = machinery.OverwriteFile

	log.Println("creating import for %", f.Resource.Path)
	f.TemplateBody = controllerTestTemplate

	return nil
}

const controllerTestTemplate = `{{ .Boilerplate }}

package {{ if and .MultiGroup .Resource.Group }}{{ .Resource.PackageName }}{{ else }}{{ .PackageName }}{{ end }}

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	{{ if not (isEmptyStr .Resource.Path) -}}
	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
	{{- end }}
)

var _ = Describe("{{ .Resource.Kind }} controller", func() {
	Context("{{ .Resource.Kind }} controller test", func() {

		const {{ .Resource.Kind }}Name = "test-{{ lower .Resource.Kind }}"

		ctx := context.Background()

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      {{ .Resource.Kind }}Name,
				Namespace: {{ .Resource.Kind }}Name,
			},
		}

		typeNamespacedName := types.NamespacedName{
			Name:      {{ .Resource.Kind }}Name,
			Namespace: {{ .Resource.Kind }}Name,
		}
		{{ lower .Resource.Kind }} := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}

		SetDefaultEventuallyTimeout(2 * time.Minute)
		SetDefaultEventuallyPollingInterval(time.Second)

		BeforeEach(func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, namespace);
			Expect(err).NotTo(HaveOccurred())

			By("Setting the Image ENV VAR which stores the Operand image")
			err= os.Setenv("{{ upper .Resource.Kind }}_IMAGE", "example.com/image:test")
			Expect(err).NotTo(HaveOccurred())

			By("creating the custom resource for the Kind {{ .Resource.Kind }}")
			err = k8sClient.Get(ctx, typeNamespacedName, {{ lower .Resource.Kind }})
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				{{ lower .Resource.Kind }} := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{
					ObjectMeta: metav1.ObjectMeta{
						Name:      {{ .Resource.Kind }}Name,
						Namespace: namespace.Name,
					},
					Spec: {{ .Resource.ImportAlias }}.{{ .Resource.Kind }}Spec{
						Size: 1,
						{{ if not (isEmptyStr .Port) -}}
						ContainerPort: {{ .Port }},
						{{- end }}
					},
				}

				err = k8sClient.Create(ctx, {{ lower .Resource.Kind }})
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterEach(func() {
			By("removing the custom resource for the Kind {{ .Resource.Kind }}")
			found := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
			err := k8sClient.Get(ctx, typeNamespacedName, found)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Delete(context.TODO(), found)).To(Succeed())
			}).Should(Succeed())

			// TODO(user): Attention if you improve this code by adding other context test you MUST
			// be aware of the current delete namespace limitations.
			// More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Deleting the Namespace to perform the tests")
			_ = k8sClient.Delete(ctx, namespace);

			By("Removing the Image ENV VAR which stores the Operand image")
			_ = os.Unsetenv("{{ upper .Resource.Kind }}_IMAGE")
		})

		It("should successfully reconcile a custom resource for {{ .Resource.Kind }}", func() {
			By("Checking if the custom resource was successfully created")
			Eventually(func(g Gomega) {
				found := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, found)).To(Succeed())
			}).Should(Succeed())

			By("Reconciling the custom resource created")
			{{ lower .Resource.Kind }}Reconciler := &{{ .Resource.Kind }}Reconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := {{ lower .Resource.Kind }}Reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func(g Gomega) {
				found := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, found)).To(Succeed())
			}).Should(Succeed())

			By("Reconciling the custom resource again")
			_, err = {{ lower .Resource.Kind }}Reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the latest Status Condition added to the {{ .Resource.Kind }} instance")
			Expect(k8sClient.Get(ctx, typeNamespacedName, {{ lower .Resource.Kind }})).To(Succeed())
			conditions := []metav1.Condition{}
			Expect({{ lower .Resource.Kind }}.Status.Conditions).To(ContainElement(
				HaveField("Type", Equal(typeAvailable{{ .Resource.Kind }})), &conditions))
			Expect(conditions).To(HaveLen(1), "Multiple conditions of type %s", typeAvailable{{ .Resource.Kind }})
			Expect(conditions[0].Status).To(Equal(metav1.ConditionTrue), "condition %s", typeAvailable{{ .Resource.Kind }})
			Expect(conditions[0].Reason).To(Equal("Reconciling"), "condition %s", typeAvailable{{ .Resource.Kind }})
		})
	})
})
`
