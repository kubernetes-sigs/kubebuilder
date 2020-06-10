/*
Copyright 2019 The Kubernetes Authors.

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

package utils

import (
	"fmt"
)

// InstallCertManager installs the cert manager bundle.
func InstallCertManager(k *Kubectl) error {
	if _, err := k.Command("create", "namespace", "cert-manager"); err != nil {
		return err
	}
	url := fmt.Sprintf(certmanagerURL, certmanagerVersion)
	if _, err := k.Apply(false, "-f", url, "--validate=false"); err != nil {
		return err
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	// was re-installed after uninstalling on a cluster.
	_, err := k.Wait(false, "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)
	return err
}

// InstallPrometheusOperManager installs the prometheus manager bundle.
func InstallPrometheusOperManager(k *Kubectl) error {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	_, err := k.Apply(false, "-f", url)
	return err
}

// UninstallPrometheusOperManager uninstalls the prometheus manager bundle.
func UninstallPrometheusOperManager(k *Kubectl) {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	if _, err := k.Delete(false, "-f", url); err != nil {
		fmt.Fprintf(GinkgoWriter, "error when running kubectl delete during cleaning up prometheus bundle: %v\n", err)
	}
}

// UninstallCertManager uninstalls the cert manager bundle.
func UninstallCertManager(k *Kubectl) {
	url := fmt.Sprintf(certmanagerURL, certmanagerVersion)
	if _, err := k.Delete(false, "-f", url); err != nil {
		fmt.Fprintf(GinkgoWriter,
			"warning: error when running kubectl delete during cleaning up cert manager: %v\n", err)
	}
	if _, err := k.Delete(false, "namespace", "cert-manager"); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when cleaning up the cert manager namespace: %v\n", err)
	}
}

