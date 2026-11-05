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

package scaffolds

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("enableCertManagerInChartWorkflow", func() {
	var testChartPath string

	BeforeEach(func() {
		testChartPath = filepath.Join(GinkgoT().TempDir(), "test-chart.yml")
	})

	writeWorkflow := func(step string) {
		content := "    steps:\n# TODO: Uncomment if cert-manager is enabled\n" + step +
			"\n\n# TODO: Uncomment if Prometheus is enabled\n      - name: Deploy manager via Helm\n"
		Expect(os.WriteFile(testChartPath, []byte(content), 0o644)).To(Succeed())
	}

	readWorkflow := func() string {
		content, err := os.ReadFile(testChartPath)
		Expect(err).NotTo(HaveOccurred())
		return string(content)
	}

	It("uncomments the step that applies the pinned cert-manager manifest", func() {
		writeWorkflow(certManagerChartWorkflowStep)

		enableCertManagerInChartWorkflow(testChartPath)

		Expect(readWorkflow()).To(ContainSubstring(
			strings.ReplaceAll(certManagerChartWorkflowStep, "\n#", "\n")[1:]))
		Expect(readWorkflow()).NotTo(ContainSubstring("#      - name: Install cert-manager"))
		Expect(readWorkflow()).To(ContainSubstring(
			"            --timeout 300s\n\n# TODO: Uncomment if Prometheus is enabled\n"))
	})

	It("uncomments the Helm chart step of workflows scaffolded before cert-manager was pinned", func() {
		writeWorkflow(legacyCertManagerChartWorkflowStep)

		enableCertManagerInChartWorkflow(testChartPath)

		Expect(readWorkflow()).To(ContainSubstring(
			"      - name: Install cert-manager via Helm (wait for readiness)\n" +
				"        run: |\n" +
				"          helm repo add jetstack https://charts.jetstack.io\n"))
		Expect(readWorkflow()).NotTo(ContainSubstring("#      - name: Install cert-manager"))
	})

	It("leaves a workflow that already installs cert-manager unchanged", func() {
		writeWorkflow(strings.ReplaceAll(certManagerChartWorkflowStep, "\n#", "\n")[1:])
		before := readWorkflow()

		enableCertManagerInChartWorkflow(testChartPath)

		Expect(readWorkflow()).To(Equal(before))
	})

	It("does nothing when the workflow does not exist", func() {
		enableCertManagerInChartWorkflow(testChartPath)

		_, err := os.Stat(testChartPath)
		Expect(os.IsNotExist(err)).To(BeTrue())
	})
})
