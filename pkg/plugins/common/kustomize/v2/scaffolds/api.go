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

package scaffolds

import (
	"fmt"
	log "log/slog"
	"os"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/crd"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/rbac"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/samples"
)

var _ plugins.Scaffolder = &apiScaffolder{}

// apiScaffolder contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type apiScaffolder struct {
	config   config.Config
	resource resource.Resource

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem

	// force indicates whether to scaffold files even if they exist.
	force bool
}

// NewAPIScaffolder returns a new Scaffolder for API/controller creation operations
func NewAPIScaffolder(cfg config.Config, res resource.Resource, force bool) plugins.Scaffolder {
	return &apiScaffolder{
		config:   cfg,
		resource: res,
		force:    force,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *apiScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *apiScaffolder) Scaffold() error {
	log.Info("Writing kustomize manifests for you to edit...")

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
		machinery.WithResource(&s.resource),
	)

	// Keep track of these values before the update
	if s.resource.HasAPI() {
		if err := scaffold.Execute(
			&samples.CRDSample{Force: s.force},
			&rbac.CRDAdminRole{},
			&rbac.CRDEditorRole{},
			&rbac.CRDViewerRole{},
			&crd.Kustomization{},
			&crd.KustomizeConfig{},
		); err != nil {
			return fmt.Errorf("error scaffolding kustomize API manifests: %w", err)
		}

		// If the gvk is non-empty
		if s.resource.Group != "" || s.resource.Version != "" || s.resource.Kind != "" {
			if err := scaffold.Execute(&samples.Kustomization{}); err != nil {
				return fmt.Errorf("error scaffolding manifests: %w", err)
			}
		}

		err := pluginutil.UncommentCode(kustomizeFilePath, "#- ../crd", `#`)
		if err != nil {
			hasCRUncommented, errCheck := pluginutil.HasFileContentWith(kustomizeFilePath, "- ../crd")
			if !hasCRUncommented || errCheck != nil {
				log.Error("unable to find the target #- ../crd to uncomment in the file",
					"file_path", kustomizeFilePath)
			}
		}

		comment := fmt.Sprintf(adminEditViewRulesCommentFragment, s.config.GetProjectName())

		// Add scaffolded CRD Admin, Editor and Viewer roles in config/rbac/kustomization.yaml
		rbacKustomizeFilePath := "config/rbac/kustomization.yaml"
		// When another top-level key follows "resources:", place the comment at the
		// end of the resources list so the role entries added after it stay in it
		if content, errRead := os.ReadFile(rbacKustomizeFilePath); errRead == nil &&
			!strings.Contains(string(content), comment) {
			if end := resourcesListEnd(string(content)); end >= 0 {
				updated := string(content[:end]) + "\n" + comment + string(content[end:])
				//nolint:gosec // same permissions used by the pluginutil helpers
				if errWrite := os.WriteFile(rbacKustomizeFilePath, []byte(updated), 0o644); errWrite != nil {
					log.Error("failed to add the admin/edit/view roles comment in the file",
						"file_path", rbacKustomizeFilePath)
				}
			}
		}
		err = pluginutil.AppendCodeIfNotExist(rbacKustomizeFilePath,
			comment)
		if err != nil {
			log.Error("failed to append the admin/edit/view roles comment in the file",
				"file_path", rbacKustomizeFilePath)
		}
		crdName := strings.ToLower(s.resource.Kind)
		if s.config.IsMultiGroup() && s.resource.Group != "" {
			crdName = strings.ToLower(s.resource.Group) + "_" + crdName
		}
		// Match the indentation already used by the list, so files reformatted
		// by tools such as yamlfmt or prettier stay valid YAML
		indent := ""
		if content, errRead := os.ReadFile(rbacKustomizeFilePath); errRead == nil {
			indent = resourcesItemIndent(string(content))
		}
		err = pluginutil.InsertCodeIfNotExist(rbacKustomizeFilePath, comment,
			fmt.Sprintf("\n%[2]s- %[1]s_admin_role.yaml\n%[2]s- %[1]s_editor_role.yaml\n%[2]s- %[1]s_viewer_role.yaml",
				crdName, indent))
		if err != nil {
			log.Error("failed to add admin, editor and viewer roles in the file",
				"file_path", rbacKustomizeFilePath)
		}
		// Add an empty line at the end of the file
		err = pluginutil.AppendCodeIfNotExist(rbacKustomizeFilePath,
			`

`)
		if err != nil {
			log.Error("failed to append empty line at the end of the file",
				"file_path", rbacKustomizeFilePath)
		}
	}

	return nil
}

// resourcesItemIndent returns the leading whitespace used by the items of the
// top-level "resources:" list in a kustomization file. It returns an empty
// string when the list items are not indented or the list cannot be found.
func resourcesItemIndent(content string) string {
	inResources := false
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !inResources {
			inResources = isTopLevelResourcesKey(line)
			continue
		}
		if isListItem(trimmed) {
			return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		}
		// The first non-comment line after "resources:" is not a list item
		return ""
	}
	return ""
}

// resourcesListEnd returns the byte offset of the end of the last line of the
// top-level "resources:" list when another top-level key follows it. It returns
// -1 when the list is the last top-level key or cannot be found, in which case
// content appended at the end of the file is still part of the list.
func resourcesListEnd(content string) int {
	offset := 0
	inResources := false
	end := -1
	for line := range strings.SplitSeq(content, "\n") {
		lineEnd := offset + len(line)
		offset = lineEnd + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		topLevel := line == strings.TrimLeft(line, " \t")
		if !inResources {
			if isTopLevelResourcesKey(line) {
				inResources = true
				end = lineEnd
			}
			continue
		}
		if !topLevel || isListItem(trimmed) {
			end = lineEnd
			continue
		}
		return end
	}
	return -1
}

// isTopLevelResourcesKey reports whether the line is the top-level "resources:"
// key, optionally followed by a comment
func isTopLevelResourcesKey(line string) bool {
	if line != strings.TrimLeft(line, " \t") {
		return false
	}
	key, _, _ := strings.Cut(line, " #")
	return strings.TrimSpace(key) == "resources:"
}

// isListItem reports whether the trimmed line starts a YAML sequence item,
// including a bare "-" whose value is on the following lines
func isListItem(trimmed string) bool {
	return trimmed == "-" || strings.HasPrefix(trimmed, "- ")
}

const adminEditViewRulesCommentFragment = `# For each CRD, "Admin", "Editor" and "Viewer" roles are scaffolded by
# default, aiding admins in cluster management. Those roles are
# not used by the %s itself. You can comment the following lines
# if you do not want those helpers be installed with your Project.`
