/*
Copyright 2018 The Kubernetes Authors.

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

package installer

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/webhook/internal"
)

type AdmissionWebhookManager struct {
	// Each installer manages an webhook.AdmissionServer
	Installers []*AdmissionWebhookInstaller
}

// LoadConfig loads the configuration from configMaps for all of the admission webhook servers.
func (awi *AdmissionWebhookManager) LoadConfig() error {
	// Load configmaps that have a certain label, e.g. admissionWebhookManager:true
	// Use the configmaps to initiate the admissionWebhookInstallers
	return nil
}

// Run manages the lifecycle of the secret and webhook configuration for admission webhook servers.
// It ensures the resources exist in case a user deletes them.
// It watches the configMaps with a certain label, e.g. admissionWebhookManager:true and
// responds when there is a change.
func (awm *AdmissionWebhookManager) Run() error {
	return nil
}

// AdmissionWebhookInstaller maps to an admission webhook server.
type AdmissionWebhookInstaller struct {
	Config *internal.AdmissionWebhookInstallConfig
}

// InstallCert creates a secret with the certificate.
func (awi *AdmissionWebhookInstaller) InstallCert() error {
	//_, _, _, err := awi.Config.CertProvider.GetServerCert()
	// Set ownerRef for the secret.
	return nil
}

// UninstallCert uninstalls the secret containing the certificate.
func (awi *AdmissionWebhookInstaller) UninstallCert() error {
	return nil
}

// InstallWebhooks installs or upgrades the k8s webhook configuration.
// It respects the awi.Config.RegistrationDelay by waiting for a period of time
// before creating or upgrading the k8s webhook configuration objects.
func (awi *AdmissionWebhookInstaller) InstallWebhooks() error {
	return nil
}

// UninstallWebhooks uninstalls the webhooks managed by this installer.
func (awi *AdmissionWebhookInstaller) UninstallWebhooks() error {
	return nil
}
