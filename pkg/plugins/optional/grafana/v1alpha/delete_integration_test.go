// Copyright 2026 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build integration

package v1alpha

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("Grafana Plugin Delete", func() {
	var kbc *utils.TestContext

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())

		err = kbc.Init("--plugins", "go/v4", "--domain", kbc.Domain, "--skip-go-version-check")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		kbc.Destroy()
	})

	It("should completely undo grafana edit - before state equals after delete", func() {
		By("verifying baseline PROJECT file is clean")
		projectPath := filepath.Join(kbc.Dir, "PROJECT")
		projectBefore, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectBefore)).NotTo(ContainSubstring("grafana.kubebuilder.io/v1-alpha"),
			"baseline PROJECT should not contain grafana plugin config")

		By("adding grafana manifests")
		err = kbc.Edit("--plugins=grafana/v1-alpha")
		Expect(err).NotTo(HaveOccurred())

		By("verifying grafana files were created")
		Expect(kbc.HasFile("grafana/controller-runtime-metrics.json")).To(BeTrue())
		Expect(kbc.HasFile("grafana/custom-metrics/config.yaml")).To(BeTrue())

		By("verifying PROJECT file was updated")
		projectAfterCreate, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectAfterCreate)).To(ContainSubstring("grafana.kubebuilder.io/v1-alpha"))

		By("deleting grafana manifests")
		err = kbc.Delete("--plugins=grafana/v1-alpha")
		Expect(err).NotTo(HaveOccurred())

		By("verifying grafana files were deleted")
		Expect(kbc.HasFile("grafana/controller-runtime-metrics.json")).To(BeFalse())
		Expect(kbc.HasFile("grafana/custom-metrics/config.yaml")).To(BeFalse())
		Expect(kbc.HasFile("grafana")).To(BeFalse())

		By("verifying PROJECT file matches initial state")
		projectAfterDelete, err := os.ReadFile(projectPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(projectAfterDelete)).NotTo(ContainSubstring("grafana.kubebuilder.io/v1-alpha"),
			"grafana plugin config should be removed")
	})
})
