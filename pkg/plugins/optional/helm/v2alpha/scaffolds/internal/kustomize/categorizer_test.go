/*
Copyright 2026 The Kubernetes Authors.

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

package kustomize

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	labelControlPlaneKey   = "control-plane"
	labelControlPlaneValue = "controller-manager"
)

func makeDeployment(name string, labels map[string]string) *unstructured.Unstructured {
	d := &unstructured.Unstructured{}
	d.SetAPIVersion("apps/v1")
	d.SetKind("Deployment")
	d.SetName(name)
	if labels != nil {
		d.SetLabels(labels)
	}
	return d
}

var _ = Describe("ResourceCategorizer", func() {
	Describe("CategorizeByFunction", func() {
		It("should place the manager deployment in the manager group", func() {
			resources := &ParsedResources{
				Deployment: makeDeployment("project-v4-controller-manager", map[string]string{
					labelControlPlaneKey: labelControlPlaneValue,
				}),
			}
			groups := NewResourceCategorizer(resources).CategorizeByFunction()
			Expect(groups["manager"]).To(HaveLen(1))
			Expect(groups["manager"][0].GetName()).To(Equal("project-v4-controller-manager"))
			Expect(groups["extras"]).To(BeNil())
		})

		It("should place extra deployments in the extras group, not the manager group", func() {
			resources := &ParsedResources{
				Deployment: makeDeployment("project-v4-controller-manager", map[string]string{
					labelControlPlaneKey: labelControlPlaneValue,
				}),
				ExtraDeployments: []*unstructured.Unstructured{
					makeDeployment("project-v4-some-operator", nil),
				},
			}
			groups := NewResourceCategorizer(resources).CategorizeByFunction()

			Expect(groups["manager"]).To(HaveLen(1))
			Expect(groups["manager"][0].GetName()).To(Equal("project-v4-controller-manager"))

			Expect(groups["extras"]).To(HaveLen(1))
			Expect(groups["extras"][0].GetName()).To(Equal("project-v4-some-operator"))
		})

		It("should place multiple extra deployments in the extras group", func() {
			resources := &ParsedResources{
				Deployment: makeDeployment("controller-manager", map[string]string{
					labelControlPlaneKey: labelControlPlaneValue,
				}),
				ExtraDeployments: []*unstructured.Unstructured{
					makeDeployment("operator-a", nil),
					makeDeployment("operator-b", nil),
				},
			}
			groups := NewResourceCategorizer(resources).CategorizeByFunction()

			Expect(groups["manager"]).To(HaveLen(1))
			Expect(groups["extras"]).To(HaveLen(2))
			extraNames := []string{
				groups["extras"][0].GetName(),
				groups["extras"][1].GetName(),
			}
			Expect(extraNames).To(ConsistOf("operator-a", "operator-b"))
		})

		It("should produce no manager or extras group when resources are empty", func() {
			resources := &ParsedResources{}
			groups := NewResourceCategorizer(resources).CategorizeByFunction()
			Expect(groups["manager"]).To(BeNil())
			Expect(groups["extras"]).To(BeNil())
		})
	})
})
