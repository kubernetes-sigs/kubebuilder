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

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
)

var _ plugins.Scaffolder = &deleteAPIScaffolder{}

// deleteAPIScaffolder removes kustomize files created for an API
type deleteAPIScaffolder struct {
	config   config.Config
	resource resource.Resource
	fs       machinery.Filesystem
}

// NewDeleteAPIScaffolder returns a new scaffolder for API deletion operations
func NewDeleteAPIScaffolder(cfg config.Config, res resource.Resource) plugins.Scaffolder {
	return &deleteAPIScaffolder{
		config:   cfg,
		resource: res,
	}
}

// InjectFS implements Scaffolder
func (s *deleteAPIScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements Scaffolder - deletes kustomize files for the API
func (s *deleteAPIScaffolder) Scaffold() error {
	log.Info("Cleaning up kustomize API files...")

	kindLower := strings.ToLower(s.resource.Kind)
	multigroup := s.config.IsMultiGroup()

	// Delete sample file
	sampleFile := filepath.Join("config", "samples",
		fmt.Sprintf("%s_%s_%s.yaml", s.resource.Group, s.resource.Version, kindLower))
	if err := removeFileIfExists(s.fs.FS, sampleFile); err != nil {
		log.Warn("Failed to delete sample file", "file", sampleFile, "error", err)
	}

	// Remove sample entry from config/samples/kustomization.yaml
	s.removeSampleFromKustomization(sampleFile)

	// Delete RBAC files (best effort)
	var rbacFiles []string
	if multigroup && s.resource.Group != "" {
		rbacFiles = []string{
			filepath.Join("config", "rbac", fmt.Sprintf("%s_%s_admin_role.yaml", s.resource.Group, kindLower)),
			filepath.Join("config", "rbac", fmt.Sprintf("%s_%s_editor_role.yaml", s.resource.Group, kindLower)),
			filepath.Join("config", "rbac", fmt.Sprintf("%s_%s_viewer_role.yaml", s.resource.Group, kindLower)),
		}
	} else {
		rbacFiles = []string{
			filepath.Join("config", "rbac", fmt.Sprintf("%s_admin_role.yaml", kindLower)),
			filepath.Join("config", "rbac", fmt.Sprintf("%s_editor_role.yaml", kindLower)),
			filepath.Join("config", "rbac", fmt.Sprintf("%s_viewer_role.yaml", kindLower)),
		}
	}

	for _, rbacFile := range rbacFiles {
		if err := removeFileIfExists(s.fs.FS, rbacFile); err != nil {
			log.Warn("Failed to delete RBAC file", "file", rbacFile, "error", err)
		}
	}

	// Remove RBAC entries from config/rbac/kustomization.yaml
	s.removeRBACFromKustomization(rbacFiles)

	// Remove CRD entry from config/crd/kustomization.yaml
	s.removeCRDFromKustomization()

	// Check if this is the last API and clean up shared kustomize files
	if s.isLastAPI() {
		s.cleanupLastAPIKustomizeFiles()
	}

	return nil
}

// isLastAPI checks if this is the last API in the project
func (s *deleteAPIScaffolder) isLastAPI() bool {
	resources, err := s.config.GetResources()
	if err != nil {
		return false
	}

	for _, res := range resources {
		if res.Group == s.resource.Group && res.Version == s.resource.Version && res.Kind == s.resource.Kind {
			continue
		}
		if res.API != nil {
			return false
		}
	}

	return true
}

// removeSampleFromKustomization removes the sample entry from config/samples/kustomization.yaml
func (s *deleteAPIScaffolder) removeSampleFromKustomization(sampleFile string) {
	kustomizationPath := filepath.Join("config", "samples", "kustomization.yaml")

	// Extract just the filename from the full path
	_, filename := filepath.Split(sampleFile)
	lineToRemove := fmt.Sprintf("- %s", filename)

	removed, err := removeLinesFromKustomization(s.fs.FS, kustomizationPath, []string{lineToRemove})
	if err != nil {
		log.Warn("Failed to remove sample from kustomization", "file", kustomizationPath, "error", err)
	} else if removed {
		log.Info("Removed sample entry from kustomization", "file", kustomizationPath, "entry", lineToRemove)
	}
}

// removeRBACFromKustomization removes RBAC entries from config/rbac/kustomization.yaml
func (s *deleteAPIScaffolder) removeRBACFromKustomization(rbacFiles []string) {
	kustomizationPath := filepath.Join("config", "rbac", "kustomization.yaml")

	linesToRemove := make([]string, 0, len(rbacFiles))
	for _, rbacFile := range rbacFiles {
		// Extract just the filename from the full path
		_, filename := filepath.Split(rbacFile)
		linesToRemove = append(linesToRemove, fmt.Sprintf("- %s", filename))
	}

	removed, err := removeLinesFromKustomization(s.fs.FS, kustomizationPath, linesToRemove)
	if err != nil {
		log.Warn("Failed to remove RBAC from kustomization", "file", kustomizationPath, "error", err)
	} else if removed {
		log.Info("Removed RBAC entries from kustomization", "file", kustomizationPath)
	}
}

// removeCRDFromKustomization removes the CRD entry from config/crd/kustomization.yaml
func (s *deleteAPIScaffolder) removeCRDFromKustomization() {
	kustomizationPath := filepath.Join("config", "crd", "kustomization.yaml")

	// Construct the CRD filename based on resource
	// Format: bases/<group>_<plural>.yaml
	crdFile := fmt.Sprintf("bases/%s_%s.yaml", s.resource.QualifiedGroup(), s.resource.Plural)
	lineToRemove := fmt.Sprintf("- %s", crdFile)

	removed, err := removeLinesFromKustomization(s.fs.FS, kustomizationPath, []string{lineToRemove})
	if err != nil {
		log.Warn("Failed to remove CRD from kustomization", "file", kustomizationPath, "error", err)
	} else if removed {
		log.Info("Removed CRD entry from kustomization", "file", kustomizationPath, "entry", lineToRemove)
	}
}

// cleanupLastAPIKustomizeFiles removes CRD and sample kustomization files
// This is called ONLY when deleting the very last API in the project
func (s *deleteAPIScaffolder) cleanupLastAPIKustomizeFiles() {
	log.Info("This is the last API - removing kustomization base files")

	kustomizeFiles := []string{
		filepath.Join("config", "crd", "kustomization.yaml"),
		filepath.Join("config", "crd", "kustomizeconfig.yaml"),
		filepath.Join("config", "samples", "kustomization.yaml"),
	}

	for _, file := range kustomizeFiles {
		if err := removeFileIfExists(s.fs.FS, file); err != nil {
			log.Warn("Failed to delete kustomization file", "file", file, "error", err)
		} else {
			log.Info("Deleted kustomization file", "file", file)
		}
	}
}
