/*
Copyright 2020 The Kubernetes Authors.

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

package v4

import (
	"fmt"
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Run e2e tests using the Ginkgo runner.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting kubebuilder suite\n")
	RunSpecs(t, "Kubebuilder e2e suite")
}

// BeforeSuite run before any specs are run to perform the required actions for all e2e Go tests.
var _ = BeforeSuite(func() {
	var err error

	kbc, err := utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
	Expect(err).NotTo(HaveOccurred())
	Expect(kbc.Prepare()).To(Succeed())

	By("installing the cert-manager bundle")
	Expect(kbc.InstallCertManager()).To(Succeed())

	By("installing the Prometheus operator")
	Expect(kbc.InstallPrometheusOperManager()).To(Succeed())
})

// AfterSuite run after all the specs have run, regardless of whether any tests have failed to ensures that
// all be cleaned up
var _ = AfterSuite(func() {
	kbc, err := utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
	Expect(err).NotTo(HaveOccurred())
	Expect(kbc.Prepare()).To(Succeed())

	By("uninstalling the Prometheus manager bundle")
	kbc.UninstallPrometheusOperManager()

	By("uninstalling the cert-manager bundle")
	kbc.UninstallCertManager()
})
