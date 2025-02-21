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
// isPrometheusOperatorAlreadyInstalled will be set true when prometheus CRDs be found on the cluster
isPrometheusOperatorAlreadyInstalled = false
`

const beforeSuitePrometheus = `
By("Ensure that Prometheus is enabled")
	_ = utils.UncommentCode("config/default/kustomization.yaml", "#- ../prometheus", "#")
`

const afterSuitePrometheus = `
// Teardown Prometheus after the suite if it was not already installed
if !isPrometheusOperatorAlreadyInstalled {
	_, _ = fmt.Fprintf(GinkgoWriter, "Uninstalling Prometheus Operator...\n")
	utils.UninstallPrometheusOperator()
}
`

const checkPrometheusInstalled = `
// To prevent errors when tests run in environments with Prometheus already installed,
// we check for it's presence before execution.
// Setup Prometheus before the suite if not already installed
By("checking if prometheus is installed already")
isPrometheusOperatorAlreadyInstalled = utils.IsPrometheusCRDsInstalled()
if !isPrometheusOperatorAlreadyInstalled {
	_, _ = fmt.Fprintf(GinkgoWriter, "Installing Prometheus Operator...\n")
	Expect(utils.InstallPrometheusOperator()).To(Succeed(), "Failed to install Prometheus Operator")
} else {
	_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: Prometheus Operator is already installed. Skipping installation...\n")
}
`
const serviceMonitorE2e = `

By("validating that the ServiceMonitor for Prometheus is applied in the namespace")
	cmd = exec.Command("kubectl", "get", "ServiceMonitor", "-n", namespace)
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "ServiceMonitor should exist")

`
