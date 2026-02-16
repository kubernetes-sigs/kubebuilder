/*
Copyright 2025 The Kubernetes Authors.

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

package all

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// Run unified e2e tests using the Ginkgo runner.
// This consolidates all plugin tests (v4, helm, deployimage) into a single suite
// to share infrastructure setup (cert-manager, Prometheus) and reduce overhead.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting unified Kubebuilder e2e suite (all plugins)\n")
	RunSpecs(t, "Kubebuilder Unified E2E Suite")
}

// BeforeSuite runs once before all specs to set up shared infrastructure.
// This is run ONCE instead of 3 times (once per plugin), significantly reducing test time.
var _ = BeforeSuite(func() {
	var err error

	_, _ = fmt.Fprintf(GinkgoWriter, "\n=== Setting up shared test infrastructure ===\n")

	kbc, err := utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
	Expect(err).NotTo(HaveOccurred())
	Expect(kbc.Prepare()).To(Succeed())

	By("installing cert-manager bundle (shared across all tests)")
	Expect(kbc.InstallCertManager()).To(Succeed())

	By("installing Prometheus operator (shared across all tests)")
	Expect(kbc.InstallPrometheusOperManager()).To(Succeed())

	_, _ = fmt.Fprintf(GinkgoWriter, "=== Shared infrastructure ready ===\n\n")
})

// AfterSuite runs once after all specs to clean up shared infrastructure.
var _ = AfterSuite(func() {
	_, _ = fmt.Fprintf(GinkgoWriter, "\n=== Cleaning up shared test infrastructure ===\n")

	kbc, err := utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
	Expect(err).NotTo(HaveOccurred())
	Expect(kbc.Prepare()).To(Succeed())

	By("uninstalling Prometheus operator")
	kbc.UninstallPrometheusOperManager()

	By("uninstalling cert-manager bundle")
	kbc.UninstallCertManager()

	_, _ = fmt.Fprintf(GinkgoWriter, "=== Cleanup complete ===\n")
})
