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
// nolint:maligned
type ControllerTest struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin

	Port        string
	PackageName string
}

// SetTemplateDefaults implements file.Template
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

//nolint:lll
const controllerTestTemplate = `{{ .Boilerplate }}

package {{ if and .MultiGroup .Resource.Group }}{{ .Resource.PackageName }}{{ else }}{{ .PackageName }}{{ end }}

import (
	"context"
	"os"
	"time"
	"fmt"

	//nolint:golint
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

		typeNamespaceName := types.NamespacedName{
			Name:      {{ .Resource.Kind }}Name,
			Namespace: {{ .Resource.Kind }}Name,
		}
		{{ lower .Resource.Kind }} := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}

		BeforeEach(func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, namespace);
			Expect(err).To(Not(HaveOccurred()))

			By("Setting the Image ENV VAR which stores the Operand image")
			err= os.Setenv("{{ upper .Resource.Kind }}_IMAGE", "example.com/image:test")
			Expect(err).To(Not(HaveOccurred()))

			By("creating the custom resource for the Kind {{ .Resource.Kind }}")
			err = k8sClient.Get(ctx, typeNamespaceName, {{ lower .Resource.Kind }})
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
				Expect(err).To(Not(HaveOccurred()))
			}
		})

		AfterEach(func() {
			By("removing the custom resource for the Kind {{ .Resource.Kind }}")
			found := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
			err := k8sClient.Get(ctx, typeNamespaceName, found)
			Expect(err).To(Not(HaveOccurred()))

			Eventually(func() error {
				return k8sClient.Delete(context.TODO(), found)
			}, 2*time.Minute, time.Second).Should(Succeed())

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
			Eventually(func() error {
				found := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
				return k8sClient.Get(ctx, typeNamespaceName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Reconciling the custom resource created")
			{{ lower .Resource.Kind }}Reconciler := &{{ .Resource.Kind }}Reconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := {{ lower .Resource.Kind }}Reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespaceName,
			})
			Expect(err).To(Not(HaveOccurred()))

			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func() error {
				found := &appsv1.Deployment{}
				return k8sClient.Get(ctx, typeNamespaceName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking the latest Status Condition added to the {{ .Resource.Kind }} instance")
			Eventually(func() error {
				if {{ lower .Resource.Kind }}.Status.Conditions != nil &&
					len({{ lower .Resource.Kind }}.Status.Conditions) != 0 {
					latestStatusCondition := {{ lower .Resource.Kind }}.Status.Conditions[len({{ lower .Resource.Kind }}.Status.Conditions)-1]
					expectedLatestStatusCondition := metav1.Condition{
						Type:    typeAvailable{{ .Resource.Kind }},
						Status:  metav1.ConditionTrue,
						Reason:  "Reconciling",
						Message: fmt.Sprintf(
							"Deployment for custom resource (%s) with %d replicas created successfully", 
							{{ lower .Resource.Kind }}.Name,
							{{ lower .Resource.Kind }}.Spec.Size),
					}
					if latestStatusCondition != expectedLatestStatusCondition {
						return fmt.Errorf("The latest status condition added to the {{ .Resource.Kind }} instance is not as expected")
					}
				}
				return nil
			}, time.Minute, time.Second).Should(Succeed())
		})
	})
})
`
