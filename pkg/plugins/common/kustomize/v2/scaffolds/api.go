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
	"strings"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/crd"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
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
	log.Println("Writing kustomize manifests for you to edit...")

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
				log.Errorf("Unable to find the target #- ../crd to uncomment in the file "+
					"%s.", kustomizeFilePath)
			}
		}

		comment := fmt.Sprintf(adminEditViewRulesCommentFragment, s.config.GetProjectName())

		// Add scaffolded CRD Admin, Editor and Viewer roles in config/rbac/kustomization.yaml
		rbacKustomizeFilePath := "config/rbac/kustomization.yaml"
		err = pluginutil.AppendCodeIfNotExist(rbacKustomizeFilePath,
			comment)
		if err != nil {
			log.Errorf("Unable to append the admin/edit/view roles comment in the file "+
				"%s.", rbacKustomizeFilePath)
		}
		crdName := strings.ToLower(s.resource.Kind)
		if s.config.IsMultiGroup() && s.resource.Group != "" {
			crdName = strings.ToLower(s.resource.Group) + "_" + crdName
		}
		err = pluginutil.InsertCodeIfNotExist(rbacKustomizeFilePath, comment,
			fmt.Sprintf("\n- %[1]s_admin_role.yaml\n- %[1]s_editor_role.yaml\n- %[1]s_viewer_role.yaml", crdName))
		if err != nil {
			log.Errorf("Unable to add Admin, Editor and Viewer roles in the file "+
				"%s.", rbacKustomizeFilePath)
		}
		// Add an empty line at the end of the file
		err = pluginutil.AppendCodeIfNotExist(rbacKustomizeFilePath,
			`

`)
		if err != nil {
			log.Errorf("Unable to append empty line at the end of the file"+
				"%s.", rbacKustomizeFilePath)
		}
	}

	return nil
}

const adminEditViewRulesCommentFragment = `# For each CRD, "Admin", "Editor" and "Viewer" roles are scaffolded by
# default, aiding admins in cluster management. Those roles are
# not used by the %s itself. You can comment the following lines
# if you do not want those helpers be installed with your Project.`
