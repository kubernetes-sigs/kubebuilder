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

var _ = Describe("generateFileName", func() {
	var generator TemplatesGenerator

	BeforeEach(func() {
		generator = TemplatesGenerator{}
	})

	It("prefixes extras ServiceAccounts with kind and namespace to avoid collisions", func() {
		sa := &unstructured.Unstructured{}
		sa.SetKind("ServiceAccount")
		sa.SetName("shared-sa")
		sa.SetNamespace("team-a")

		Expect(generator.generateFileName(sa, 0, "extras", "test-project", "test-project-system")).
			To(Equal("serviceaccount-shared-sa-team-a.yaml"))
	})

	It("distinguishes extras resources with the same name in different namespaces", func() {
		saA := &unstructured.Unstructured{}
		saA.SetKind("ServiceAccount")
		saA.SetName("shared-sa")
		saA.SetNamespace("team-a")

		saB := &unstructured.Unstructured{}
		saB.SetKind("ServiceAccount")
		saB.SetName("shared-sa")
		saB.SetNamespace("team-b")

		nameA := generator.generateFileName(saA, 0, "extras", "test-project", "test-project-system")
		nameB := generator.generateFileName(saB, 0, "extras", "test-project", "test-project-system")

		Expect(nameA).NotTo(Equal(nameB))
		Expect(nameA).To(Equal("serviceaccount-shared-sa-team-a.yaml"))
		Expect(nameB).To(Equal("serviceaccount-shared-sa-team-b.yaml"))
	})

	It("prefixes extras Deployments with kind to avoid colliding with ServiceAccounts", func() {
		sa := &unstructured.Unstructured{}
		sa.SetKind("ServiceAccount")
		sa.SetName("worker")
		sa.SetNamespace("ops")

		deployment := &unstructured.Unstructured{}
		deployment.SetKind("Deployment")
		deployment.SetName("worker")
		deployment.SetNamespace("ops")

		saName := generator.generateFileName(sa, 0, "extras", "test-project", "test-project-system")
		deploymentName := generator.generateFileName(deployment, 0, "extras", "test-project", "test-project-system")

		Expect(saName).To(Equal("serviceaccount-worker-ops.yaml"))
		Expect(deploymentName).To(Equal("deployment-worker-ops.yaml"))
		Expect(saName).NotTo(Equal(deploymentName))
	})

	It("keeps non-extras RBAC filenames unchanged for manager namespace resources", func() {
		role := &unstructured.Unstructured{}
		role.SetKind("Role")
		role.SetName("test-project-leader-election")
		role.SetNamespace("test-project-system")

		Expect(generator.generateFileName(role, 0, "rbac", "test-project", "test-project-system")).
			To(Equal("leader-election.yaml"))
	})
})
