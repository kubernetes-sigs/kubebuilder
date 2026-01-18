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

// cleanupLastAPIKustomizeFiles removes CRD and sample kustomization files
func (s *deleteAPIScaffolder) cleanupLastAPIKustomizeFiles() {
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
