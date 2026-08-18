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

package appliers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("templateServiceAccountNameInBindings", func() {
	const chartName = "test-project"

	It("templates only the manager ServiceAccount subject matching name and namespace", func() {
		managerSA := &unstructured.Unstructured{}
		managerSA.SetName("shared-sa")
		managerSA.SetNamespace("test-project-system")

		yamlContent := `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: manager-binding
subjects:
- kind: ServiceAccount
  name: shared-sa
  namespace: test-project-system
- kind: ServiceAccount
  name: shared-sa
  namespace: other-namespace
`

		result := templateServiceAccountNameInBindings("test-project", chartName, yamlContent, managerSA)

		Expect(result).To(ContainSubstring(
			`name: {{ include "test-project.serviceAccountName" . }}
  namespace: test-project-system`))
		Expect(result).To(ContainSubstring(`name: shared-sa
  namespace: other-namespace`))
		Expect(result).NotTo(ContainSubstring(`name: {{ include "test-project.serviceAccountName" . }}
  namespace: other-namespace`))
	})

	It("does not template unrelated ServiceAccount subjects that share the manager name", func() {
		managerSA := &unstructured.Unstructured{}
		managerSA.SetName("shared-sa")
		managerSA.SetNamespace("test-project-system")

		yamlContent := `subjects:
- kind: ServiceAccount
  name: shared-sa
  namespace: external
`

		result := templateServiceAccountNameInBindings("test-project", chartName, yamlContent, managerSA)
		Expect(result).To(Equal(yamlContent))
	})
})

var _ = Describe("templateServiceAccountName", func() {
	const chartName = "test-project"

	It("templates metadata.name without matching metadata.labels keys", func() {
		managerSA := &unstructured.Unstructured{}
		managerSA.SetName("shared-sa")

		yamlContent := `apiVersion: v1
kind: ServiceAccount
metadata:
  name: shared-sa
  labels:
    app.kubernetes.io/name: shared-sa
`

		result := templateServiceAccountName("test-project", chartName, yamlContent, managerSA)

		Expect(result).To(ContainSubstring(
			`name: {{ include "test-project.serviceAccountName" . }}`))
		Expect(result).To(ContainSubstring(`app.kubernetes.io/name: shared-sa`))
		Expect(result).NotTo(ContainSubstring(`app.kubernetes.io/name: {{ include "test-project.serviceAccountName" . }}`))
	})
})
