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
	"errors"
	"fmt"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo" //nolint:golint
)

// Kubectl contains context to run kubectl commands
type Kubectl struct {
	*CmdContext
	Namespace string
}

// Command is a general func to run kubectl commands
func (k *Kubectl) Command(cmdOptions ...string) (string, error) {
	cmd := exec.Command("kubectl", cmdOptions...)
	output, err := k.Output(cmd)
	return string(output), err
}

// WithInput is a general func to run kubectl commands with input
func (k *Kubectl) WithInput(stdinInput string) *Kubectl {
	k.Stdin = strings.NewReader(stdinInput)
	return k
}

// CommandInNamespace is a general func to run kubectl commands in the namespace
func (k *Kubectl) CommandInNamespace(cmdOptions ...string) (string, error) {
	if len(k.Namespace) == 0 {
		return "", errors.New("namespace should not be empty")
	}
	return k.Command(append([]string{"-n", k.Namespace}, cmdOptions...)...)
}

// Apply is a general func to run kubectl apply commands
func (k *Kubectl) Apply(inNamespace bool, cmdOptions ...string) (string, error) {
	ops := append([]string{"apply"}, cmdOptions...)
	if inNamespace {
		return k.CommandInNamespace(ops...)
	}
	return k.Command(ops...)
}

// Get is a func to run kubectl get commands
func (k *Kubectl) Get(inNamespace bool, cmdOptions ...string) (string, error) {
	ops := append([]string{"get"}, cmdOptions...)
	if inNamespace {
		return k.CommandInNamespace(ops...)
	}
	return k.Command(ops...)
}

// Delete is a func to run kubectl delete commands
func (k *Kubectl) Delete(inNamespace bool, cmdOptions ...string) (string, error) {
	ops := append([]string{"delete"}, cmdOptions...)
	if inNamespace {
		return k.CommandInNamespace(ops...)
	}
	return k.Command(ops...)
}

// Logs is a func to run kubectl logs commands
func (k *Kubectl) Logs(cmdOptions ...string) (string, error) {
	ops := append([]string{"logs"}, cmdOptions...)
	return k.CommandInNamespace(ops...)
}

// Wait is a func to run kubectl wait commands
func (k *Kubectl) Wait(inNamespace bool, cmdOptions ...string) (string, error) {
	ops := append([]string{"wait"}, cmdOptions...)
	if inNamespace {
		return k.CommandInNamespace(ops...)
	}
	return k.Command(ops...)
}

// InstallCertManager installs the cert manager bundle.
func (k *Kubectl) InstallCertManager() error {
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
func (k *Kubectl) InstallPrometheusOperManager() error {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	_, err := k.Apply(false, "-f", url)
	return err
}

// UninstallPrometheusOperManager uninstalls the prometheus manager bundle.
func (k *Kubectl) UninstallPrometheusOperManager() {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	if _, err := k.Delete(false, "-f", url); err != nil {
		fmt.Fprintf(GinkgoWriter, "error when running kubectl delete during cleaning up prometheus bundle: %v\n", err)
	}
}

// UninstallCertManager uninstalls the cert manager bundle.
func (k *Kubectl) UninstallCertManager() {
	url := fmt.Sprintf(certmanagerURL, certmanagerVersion)
	if _, err := k.Delete(false, "-f", url); err != nil {
		fmt.Fprintf(GinkgoWriter,
			"warning: error when running kubectl delete during cleaning up cert manager: %v\n", err)
	}
	if _, err := k.Delete(false, "namespace", "cert-manager"); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when cleaning up the cert manager namespace: %v\n", err)
	}
}
