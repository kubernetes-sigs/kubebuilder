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

package common

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("IsManagerDeployment", func() {
	It("returns false for nil", func() {
		Expect(IsManagerDeployment(nil)).To(BeFalse())
	})

	It("returns true for control-plane label", func() {
		d := &unstructured.Unstructured{}
		d.SetLabels(map[string]string{"control-plane": "controller-manager"})
		Expect(IsManagerDeployment(d)).To(BeTrue())
	})

	It("returns true when a container is named manager", func() {
		d := &unstructured.Unstructured{Object: map[string]any{}}
		Expect(unstructured.SetNestedField(
			d.Object, []any{map[string]any{"name": DefaultManagerContainerName}},
			"spec", "template", "spec", "containers",
		)).To(Succeed())
		Expect(IsManagerDeployment(d)).To(BeTrue())
	})

	It("returns true when the deployment name contains controller-manager", func() {
		d := &unstructured.Unstructured{}
		d.SetName("my-project-controller-manager")
		Expect(IsManagerDeployment(d)).To(BeTrue())
	})

	It("returns false for default-container annotation alone", func() {
		d := &unstructured.Unstructured{Object: map[string]any{}}
		Expect(unstructured.SetNestedField(
			d.Object, map[string]any{DefaultContainerAnnotation: "worker"},
			"spec", "template", "metadata", "annotations",
		)).To(Succeed())
		Expect(unstructured.SetNestedField(
			d.Object, []any{map[string]any{"name": "worker"}},
			"spec", "template", "spec", "containers",
		)).To(Succeed())
		Expect(IsManagerDeployment(d)).To(BeFalse())
	})
})
