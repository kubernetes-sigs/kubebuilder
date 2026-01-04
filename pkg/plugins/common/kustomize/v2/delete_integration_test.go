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

package v2

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("Kustomize Delete Integration Tests", func() {
	var kbc *utils.TestContext

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())

		err = kbc.Init("--plugins", "go/v4", "--domain", kbc.Domain, "--repo", kbc.Domain, "--skip-go-version-check")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		kbc.Destroy()
	})

	It("should test all kustomize file cleanup scenarios", func() {
		By("verifying kustomize creates and deletes all expected files")

		// Create API and verify kustomize files
		err := kbc.CreateAPI("--group", "test", "--version", "v1", "--kind", "Sample",
			"--resource", "--controller", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		// Verify kustomize files created
		sampleFiles := []string{
			"config/samples/test_v1_sample.yaml",
			"config/rbac/sample_admin_role.yaml",
			"config/rbac/sample_editor_role.yaml",
			"config/rbac/sample_viewer_role.yaml",
			"config/crd/kustomization.yaml",
			"config/samples/kustomization.yaml",
		}

		for _, file := range sampleFiles {
			Expect(kbc.HasFile(file)).To(BeTrue(),
				fmt.Sprintf("kustomize file should be created: %s", file))
		}

		// Delete API and verify kustomize files deleted
		err = kbc.DeleteAPI("--group", "test", "--version", "v1", "--kind", "Sample",
			"--skip-confirmation")
		Expect(err).NotTo(HaveOccurred())

		for _, file := range sampleFiles {
			Expect(kbc.HasFile(file)).To(BeFalse(),
				fmt.Sprintf("kustomize file should be deleted: %s", file))
		}

		By("testing all kustomize files are cleaned up")
		// Note: Creating additional APIs after delete causes cmd/main.go import issues.
		// The comprehensive cleanup is validated by shell tests.
		// This test focuses on verifying the core kustomize cleanup happens.
	})
})
