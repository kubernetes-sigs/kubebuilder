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
	"fmt"
	log "log/slog"
	"path/filepath"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
)

var _ plugins.Scaffolder = &deleteWebhookScaffolder{}

// deleteWebhookScaffolder removes kustomize files created for webhooks
type deleteWebhookScaffolder struct {
	config   config.Config
	resource resource.Resource
	fs       machinery.Filesystem

	// Track which webhook types were deleted (to remove conversion patch)
	deletedConversion bool
}

// NewDeleteWebhookScaffolder returns a new scaffolder for webhook deletion operations
func NewDeleteWebhookScaffolder(cfg config.Config, res resource.Resource) *DeleteWebhookScaffolder {
	return &DeleteWebhookScaffolder{
		config:   cfg,
		resource: res,
	}
}

// DeleteWebhookScaffolder is the exported type
type DeleteWebhookScaffolder = deleteWebhookScaffolder

// SetDeletedWebhookTypes sets which webhook types were deleted
func (s *deleteWebhookScaffolder) SetDeletedWebhookTypes(conversion bool) {
	s.deletedConversion = conversion
}

// InjectFS implements Scaffolder
func (s *deleteWebhookScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements Scaffolder
func (s *deleteWebhookScaffolder) Scaffold() error {
	return nil
}

// PostScaffold cleans up kustomize files after golang plugin has updated config
func (s *deleteWebhookScaffolder) PostScaffold() error {
	log.Info("Checking kustomize webhook cleanup...")

	// If conversion webhook was deleted, clean up the webhook patch entry
	if s.deletedConversion {
		s.removeConversionWebhookPatch()

		// Check if there are still conversion webhooks in the project
		// If not, comment out the configurations section (only conversion webhooks need it uncommented)
		if !s.hasAnyConversionWebhookRemaining() {
			s.commentOutCRDKustomizeConfigurations()
		}
	}

	// Now check if this is the last webhook (after golang/v4 updated the config)
	isLastWebhook := s.isLastWebhookInProject()

	log.Info("Checking if last webhook", "isLast", isLastWebhook)

	if isLastWebhook {
		log.Info("This is the last webhook in the project, cleaning up all webhook kustomize files...")
		s.cleanupAllWebhookKustomizeFiles()
		s.commentWebhookKustomizeConfiguration()
	} else {
		log.Info("Other webhooks still exist, skipping full cleanup")
	}

	return nil
}

// isLastWebhookInProject checks if there are any webhooks remaining in the project
func (s *deleteWebhookScaffolder) isLastWebhookInProject() bool {
	resources, err := s.config.GetResources()
	if err != nil {
		return false
	}

	for _, res := range resources {
		if res.Webhooks != nil && !res.Webhooks.IsEmpty() {
			return false
		}
	}

	return true
}

// hasAnyConversionWebhookRemaining checks if any conversion webhooks still exist in the project
func (s *deleteWebhookScaffolder) hasAnyConversionWebhookRemaining() bool {
	resources, err := s.config.GetResources()
	if err != nil {
		return false
	}

	for _, res := range resources {
		if res.Webhooks != nil && res.Webhooks.Conversion {
			return true
		}
	}

	return false
}

// commentOutCRDKustomizeConfigurations comments out the configurations section in config/crd/kustomization.yaml
// This should be called when the last conversion webhook is deleted (conversion webhooks need this uncommented)
func (s *deleteWebhookScaffolder) commentOutCRDKustomizeConfigurations() {
	crdKustomizePath := filepath.Join("config", "crd", "kustomization.yaml")

	// Use exact replacement to preserve newlines precisely
	uncommented := `configurations:
- kustomizeconfig.yaml
`
	commented := `#configurations:
#- kustomizeconfig.yaml
`

	if err := util.ReplaceInFile(crdKustomizePath, uncommented, commented); err != nil {
		log.Warn("Unable to comment out configurations section in CRD kustomization",
			"file", crdKustomizePath, "error", err)
	} else {
		log.Info("Commented out configurations section in CRD kustomization (no conversion webhooks remain)",
			"file", crdKustomizePath)
	}
}

// removeConversionWebhookPatch removes the conversion webhook patch entry from config/crd/kustomization.yaml
func (s *deleteWebhookScaffolder) removeConversionWebhookPatch() {
	kustomizationPath := filepath.Join("config", "crd", "kustomization.yaml")

	// Determine the suffix based on multigroup
	multigroup := s.config.IsMultiGroup()
	suffix := s.resource.Plural
	if multigroup && s.resource.Group != "" {
		suffix = s.resource.Group + "_" + s.resource.Plural
	}

	// The patch entry format: - path: patches/webhook_in_<suffix>.yaml
	patchEntry := fmt.Sprintf("- path: patches/webhook_in_%s.yaml", suffix)

	removed, err := removeLinesFromKustomization(s.fs.FS, kustomizationPath, []string{patchEntry})
	if err != nil {
		log.Warn("Failed to remove conversion webhook patch from kustomization",
			"file", kustomizationPath, "error", err)
	} else if removed {
		log.Info("Removed conversion webhook patch entry from kustomization",
			"file", kustomizationPath, "entry", patchEntry)

		// Also delete the actual patch file
		patchFile := filepath.Join("config", "crd", "patches", fmt.Sprintf("webhook_in_%s.yaml", suffix))
		if err := removeFileIfExists(s.fs.FS, patchFile); err != nil {
			log.Warn("Failed to delete conversion webhook patch file", "file", patchFile, "error", err)
		}
	}
}

// cleanupAllWebhookKustomizeFiles removes all webhook-related kustomize files
func (s *deleteWebhookScaffolder) cleanupAllWebhookKustomizeFiles() {
	filesToDelete := []string{
		filepath.Join("config", "certmanager", "certificate.yaml"),
		filepath.Join("config", "certmanager", "certificate-webhook.yaml"),
		filepath.Join("config", "certmanager", "certificate-metrics.yaml"),
		filepath.Join("config", "certmanager", "issuer.yaml"),
		filepath.Join("config", "certmanager", "kustomization.yaml"),
		filepath.Join("config", "certmanager", "kustomizeconfig.yaml"),
		filepath.Join("config", "webhook", "kustomization.yaml"),
		filepath.Join("config", "webhook", "service.yaml"),
		filepath.Join("config", "default", "manager_webhook_patch.yaml"),
		filepath.Join("config", "network-policy", "allow-webhook-traffic.yaml"),
	}

	for _, file := range filesToDelete {
		if err := removeFileIfExists(s.fs.FS, file); err != nil {
			log.Warn("Failed to delete webhook kustomize file", "file", file, "error", err)
		}
	}

	// Delete directories if they're now empty or force delete them
	dirsToDelete := []string{
		filepath.Join("config", "certmanager"),
		filepath.Join("config", "webhook"),
	}

	for _, dir := range dirsToDelete {
		if exists, _ := afero.DirExists(s.fs.FS, dir); exists {
			if err := s.fs.FS.RemoveAll(dir); err != nil {
				log.Warn("Failed to delete directory", "dir", dir, "error", err)
			} else {
				log.Info("Deleted directory", "dir", dir)
			}
		}
	}
}

// commentWebhookKustomizeConfiguration comments out webhook-related kustomize configuration
func (s *deleteWebhookScaffolder) commentWebhookKustomizeConfiguration() {
	kustomizeFilePath := filepath.Join("config", "default", "kustomization.yaml")

	// Comment out ../webhook directory reference
	if err := util.CommentCode(kustomizeFilePath, "- ../webhook", "#"); err != nil {
		log.Warn("Unable to comment out '../webhook' in kustomization.yaml",
			"file", kustomizeFilePath, "error", err)
	}

	// Comment out ../certmanager directory reference
	if err := util.CommentCode(kustomizeFilePath, "- ../certmanager", "#"); err != nil {
		log.Warn("Unable to comment out '../certmanager' in kustomization.yaml",
			"file", kustomizeFilePath, "error", err)
	}

	// Comment out patches section header (best effort)
	if err := util.CommentCode(kustomizeFilePath, "patches:", "#"); err != nil {
		log.Warn("Unable to comment out 'patches:' section",
			"file", kustomizeFilePath, "error", err)
	}

	// Comment out manager webhook patch
	managerPatchBlock := `- path: manager_webhook_patch.yaml
  target:
    kind: Deployment`
	if err := util.CommentCode(kustomizeFilePath, managerPatchBlock, "#"); err != nil {
		log.Warn("Unable to comment out manager_webhook_patch.yaml",
			"file", kustomizeFilePath, "error", err)
	}

	// Comment out replacements section (best effort)
	if err := util.CommentCode(kustomizeFilePath, "replacements:", "#"); err != nil {
		log.Warn("Unable to comment out 'replacements:' section",
			"file", kustomizeFilePath, "error", err)
	}

	// Remove webhook traffic line from network policy
	networkPolicyPath := filepath.Join("config", "network-policy", "kustomization.yaml")
	if err := util.ReplaceInFile(networkPolicyPath, "- allow-webhook-traffic.yaml\n", ""); err != nil {
		log.Warn("Unable to remove 'allow-webhook-traffic.yaml' from network policy",
			"file", networkPolicyPath, "error", err)
	}

	// Note: CRD kustomization configurations section is handled separately in commentOutCRDKustomizeConfigurations()
	// when the last conversion webhook is deleted (called from PostScaffold when deletedConversion is true)
}
