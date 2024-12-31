/*
Copyright 2022 The Kubernetes Authors.

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

package grafana

import (
	"path/filepath"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"

	. "github.com/onsi/ginkgo/v2"

	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("kubebuilder", func() {
	Context("plugin grafana/v1-alpha", func() {
		var (
			kbc *utils.TestContext
		)

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())
		})

		AfterEach(func() {
			kbc.Destroy()
		})

		It("should generate a runnable project with grafana plugin", func() {
			GenerateProject(kbc)
		})

	})
})

// GenerateProject implements a grafana/v1(-alpha) plugin project defined by a TestContext.
func GenerateProject(kbc *utils.TestContext) {
	var err error

	By("initializing a project")
	err = kbc.Init(
		"--plugins", "grafana.kubebuilder.io/v1-alpha",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to initialize project")

	By("verifying the initial template content and updating for real custom metrics")
	ExpectWithOffset(1, pluginutil.ReplaceInFile(
		filepath.Join(kbc.Dir, "grafana", "custom-metrics", "config.yaml"),
		`---
customMetrics:
#  - metric: # Raw custom metric (required)
#    type:   # Metric type: counter/gauge/histogram (required)
#    expr:   # Prom_ql for the metric (optional)
`,
		`---
customMetrics:
  - metric: foo_bar
    type: counter
  - metric: foo_bar
    type: histogram
`)).To(Succeed())

	By("editing a project based on grafana/custom-metrics/config.yaml")
	err = kbc.Edit(
		"--plugins", "grafana.kubebuilder.io/v1-alpha",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to edit base of the project")

	fileContainsExpr, err := pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "grafana", "custom-metrics", "custom-metrics-dashboard.json"),
		`sum(rate(foo_bar{job=\"$job\", namespace=\"$namespace\"}[5m])) by (instance, pod)`)
	Expect(err).NotTo(HaveOccurred(), "Failed to edit sum rate for custom metrics")
	Expect(fileContainsExpr).To(BeTrue())

	fileContainsExpr, err = pluginutil.HasFileContentWith(
		filepath.Join(kbc.Dir, "grafana", "custom-metrics", "custom-metrics-dashboard.json"),
		`histogram_quantile(0.90, sum by(instance, le) (rate(foo_bar{job=\"$job\", namespace=\"$namespace\"}[5m])))`)
	Expect(err).NotTo(HaveOccurred(), "Failed to edit histogram_quantile for custom metrics")
	Expect(fileContainsExpr).To(BeTrue())
}
