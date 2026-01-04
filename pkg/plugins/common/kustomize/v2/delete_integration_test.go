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
	"os"
	"path/filepath"

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

		By("verifying baseline is clean (kustomize files don't exist yet)")
		Expect(kbc.HasFile("config/samples/kustomization.yaml")).To(BeFalse(),
			"baseline should not have samples kustomization")
		Expect(kbc.HasFile("config/crd/kustomization.yaml")).To(BeFalse(),
			"baseline should not have CRD kustomization")

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
			"-y")
		Expect(err).NotTo(HaveOccurred())

		for _, file := range sampleFiles {
			Expect(kbc.HasFile(file)).To(BeFalse(),
				fmt.Sprintf("kustomize file should be deleted: %s", file))
		}

		By("verifying all kustomization files are properly cleaned up")

		// Create multiple APIs to test selective kustomization entry removal
		err = kbc.CreateAPI("--group", "test", "--version", "v1", "--kind", "First",
			"--resource", "--controller", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		err = kbc.CreateAPI("--group", "test", "--version", "v1", "--kind", "Second",
			"--resource", "--controller", "--make=false")
		Expect(err).NotTo(HaveOccurred())

		By("verifying both APIs in kustomization files")
		crdKustomize, err := os.ReadFile(filepath.Join(kbc.Dir, "config", "crd", "kustomization.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(crdKustomize)).To(ContainSubstring("test." + kbc.Domain + "_firsts.yaml"))
		Expect(string(crdKustomize)).To(ContainSubstring("test." + kbc.Domain + "_seconds.yaml"))

		samplesKustomize, err := os.ReadFile(filepath.Join(kbc.Dir, "config", "samples", "kustomization.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(samplesKustomize)).To(ContainSubstring("test_v1_first.yaml"))
		Expect(string(samplesKustomize)).To(ContainSubstring("test_v1_second.yaml"))

		By("deleting first API")
		err = kbc.DeleteAPI("--group", "test", "--version", "v1", "--kind", "First", "-y")
		Expect(err).NotTo(HaveOccurred())

		By("verifying First removed but Second remains in kustomization files")
		crdKustomize, err = os.ReadFile(filepath.Join(kbc.Dir, "config", "crd", "kustomization.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(crdKustomize)).NotTo(ContainSubstring("test."+kbc.Domain+"_firsts.yaml"),
			"First should be removed from CRD kustomization")
		Expect(string(crdKustomize)).To(ContainSubstring("test."+kbc.Domain+"_seconds.yaml"),
			"Second should remain in CRD kustomization")

		samplesKustomize, err = os.ReadFile(filepath.Join(kbc.Dir, "config", "samples", "kustomization.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(samplesKustomize)).NotTo(ContainSubstring("test_v1_first.yaml"),
			"First should be removed from samples kustomization")
		Expect(string(samplesKustomize)).To(ContainSubstring("test_v1_second.yaml"),
			"Second should remain in samples kustomization")

		rbacKustomize, err := os.ReadFile(filepath.Join(kbc.Dir, "config", "rbac", "kustomization.yaml"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(rbacKustomize)).NotTo(ContainSubstring("first_admin_role.yaml"),
			"First RBAC should be removed")
		Expect(string(rbacKustomize)).To(ContainSubstring("second_admin_role.yaml"),
			"Second RBAC should remain")

		By("verifying kustomization files still exist (not last API)")
		Expect(kbc.HasFile(filepath.Join("config", "crd", "kustomization.yaml"))).To(BeTrue(),
			"CRD kustomization should exist when APIs remain")
		Expect(kbc.HasFile(filepath.Join("config", "samples", "kustomization.yaml"))).To(BeTrue(),
			"Samples kustomization should exist when APIs remain")

		By("deleting last API")
		err = kbc.DeleteAPI("--group", "test", "--version", "v1", "--kind", "Second", "-y")
		Expect(err).NotTo(HaveOccurred())

		By("verifying kustomization files deleted with last API")
		Expect(kbc.HasFile(filepath.Join("config", "crd", "kustomization.yaml"))).To(BeFalse(),
			"CRD kustomization should be deleted with last API")
		Expect(kbc.HasFile(filepath.Join("config", "crd", "kustomizeconfig.yaml"))).To(BeFalse(),
			"CRD kustomizeconfig should be deleted with last API")
		Expect(kbc.HasFile(filepath.Join("config", "samples", "kustomization.yaml"))).To(BeFalse(),
			"Samples kustomization should be deleted with last API")
	})
})
