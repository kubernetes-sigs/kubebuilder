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
	"strings"

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
}

// NewDeleteWebhookScaffolder returns a new scaffolder for webhook deletion operations
func NewDeleteWebhookScaffolder(cfg config.Config, res resource.Resource) plugins.Scaffolder {
	return &deleteWebhookScaffolder{
		config:   cfg,
		resource: res,
	}
}

// InjectFS implements Scaffolder
func (s *deleteWebhookScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements Scaffolder - does nothing during Scaffold phase
// Kustomize cleanup is done in PostScaffold after golang/v4 updates the PROJECT file
func (s *deleteWebhookScaffolder) Scaffold() error {
	// Do nothing here - cleanup happens in hasPostScaffold hook
	// This ensures golang/v4 plugin has already updated the webhooks in config
	return nil
}

// PostScaffold cleans up kustomize files after golang plugin has updated config
func (s *deleteWebhookScaffolder) PostScaffold() error {
	log.Info("Checking kustomize webhook cleanup...")

	// Now check if this is the last webhook (after golang/v4 updated the config)
	isLastWebhook := s.isLastWebhookInProject()

	log.Info("Checking if last webhook", "isLast", isLastWebhook)

	if isLastWebhook {
		log.Info("This is the last webhook in the project, cleaning up all webhook kustomize files...")
		s.cleanupAllWebhookKustomizeFiles()
		s.commentWebhookKustomizeConfiguration()
	} else {
		log.Info("Other webhooks still exist, skipping kustomize cleanup")
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
	if err := commentCodeBlock(s.fs, kustomizeFilePath, managerPatchBlock); err != nil {
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
	if err := removeLineFromFile(s.fs, networkPolicyPath, "- allow-webhook-traffic.yaml"); err != nil {
		log.Warn("Unable to remove 'allow-webhook-traffic.yaml' from network policy",
			"file", networkPolicyPath, "error", err)
	}
}

// commentCodeBlock comments out a multi-line block in a file
func commentCodeBlock(fs machinery.Filesystem, filePath, block string) error {
	content, err := afero.ReadFile(fs.FS, filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	blockLines := strings.Split(block, "\n")
	lines := strings.Split(string(content), "\n")
	modified := false
	prefix := "#"

	// Find the starting line of the block
	for i := 0; i <= len(lines)-len(blockLines); i++ {
		match := true
		for j, blockLine := range blockLines {
			if strings.TrimSpace(lines[i+j]) != strings.TrimSpace(blockLine) {
				match = false
				break
			}
		}

		if match {
			// Comment out all lines in the block
			for j := range blockLines {
				line := lines[i+j]
				if !strings.HasPrefix(strings.TrimSpace(line), prefix) {
					indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
					lines[i+j] = indent + prefix + strings.TrimLeft(line, " \t")
				}
			}
			modified = true
			break
		}
	}

	if !modified {
		return fmt.Errorf("block not found or already commented")
	}

	if err := afero.WriteFile(fs.FS, filePath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// removeLineFromFile removes a line containing the target text from a file
func removeLineFromFile(fs machinery.Filesystem, filePath, target string) error {
	content, err := afero.ReadFile(fs.FS, filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	newLines := []string{}
	removed := false

	for _, line := range lines {
		if strings.Contains(line, target) {
			removed = true
			continue
		}
		newLines = append(newLines, line)
	}

	if !removed {
		return fmt.Errorf("target %q not found in file", target)
	}

	if err := afero.WriteFile(fs.FS, filePath, []byte(strings.Join(newLines, "\n")), 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
