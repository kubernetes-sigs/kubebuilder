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

package cronjob

const isPrometheusInstalledVar = `
	// shouldCleanupPrometheus tracks whether Prometheus was installed by this suite.
	shouldCleanupPrometheus = false`

const beforeSuitePrometheus = `
	By("Ensure that Prometheus is enabled")
	_ = utils.UncommentCode("config/default/kustomization.yaml", "#- ../prometheus", "#")
`

const afterSuitePrometheus = `
	// Teardown Prometheus if it was installed by this suite
	if shouldCleanupPrometheus {
		By("uninstalling Prometheus Operator")
		utils.UninstallPrometheusOperator()
	}
`

const checkPrometheusInstalled = `
	By("checking if Prometheus is already installed")
	if !utils.IsPrometheusCRDsInstalled() {
		// Mark for cleanup before installation to handle interruptions and partial installs.
		shouldCleanupPrometheus = true

		By("installing Prometheus Operator")
		Expect(utils.InstallPrometheusOperator()).To(Succeed(), "Failed to install Prometheus Operator")
	}
`

const serviceMonitorE2e = `

			By("validating that the ServiceMonitor for Prometheus is applied in the namespace")
			cmd = exec.Command("kubectl", "get", "ServiceMonitor", "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "ServiceMonitor should exist")`

const prometheusUtilities = `// InstallPrometheusOperator installs the prometheus Operator to be used to export the enabled metrics.
func InstallPrometheusOperator() error {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "create", "-f", url)
	_, err := Run(cmd)
	return err
}

// UninstallPrometheusOperator uninstalls the prometheus
func UninstallPrometheusOperator() {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	cmd := exec.Command("kubectl", "delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}
}

// IsPrometheusCRDsInstalled checks if any Prometheus CRDs are installed
// by verifying the existence of key CRDs related to Prometheus.
func IsPrometheusCRDsInstalled() bool {
	// List of common Prometheus CRDs
	prometheusCRDs := []string{
		"prometheuses.monitoring.coreos.com",
		"prometheusrules.monitoring.coreos.com",
		"prometheusagents.monitoring.coreos.com",
	}

	cmd := exec.Command("kubectl", "get", "crds", "-o", "custom-columns=NAME:.metadata.name")
	output, err := Run(cmd)
	if err != nil {
		return false
	}
	crdList := GetNonEmptyLines(output)
	for _, crd := range prometheusCRDs {
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				return true
			}
		}
	}

	return false
}
`

const prometheusVersionURL = `

	prometheusOperatorVersion = "v0.89.0"
	prometheusOperatorURL     = "https://github.com/prometheus-operator/prometheus-operator/" +
		"releases/download/%s/bundle.yaml"`
