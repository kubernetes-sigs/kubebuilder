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
	"os"
	"strings"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/multicluster-runtime/v1alpha/scaffolds/internal/templates/cmd"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/multicluster-runtime/v1alpha/scaffolds/internal/templates/controllers"
)

var _ plugins.Scaffolder = &apiScaffolder{}

type apiScaffolder struct {
	config   config.Config
	resource resource.Resource
	fs       machinery.Filesystem
}

// NewAPIScaffolder returns a Scaffolder for the create api command.
func NewAPIScaffolder(cfg config.Config, res resource.Resource) plugins.Scaffolder {
	return &apiScaffolder{config: cfg, resource: res}
}

// InjectFS implements plugins.Scaffolder.
func (s *apiScaffolder) InjectFS(fs machinery.Filesystem) { s.fs = fs }

// Scaffold overwrites the controller and controller test files with multicluster-aware versions.
func (s *apiScaffolder) Scaffold() error {
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
		machinery.WithResource(&s.resource),
	)

	if err := scaffold.Execute(
		&controllers.Controller{Force: true},
		&controllers.ControllerTest{Force: true, DoAPI: s.resource.HasAPI()},
	); err != nil {
		return fmt.Errorf("failed to execute scaffold: %w", err)
	}

	if err := scaffold.Execute(
		&cmd.MainUpdater{WireController: true},
	); err != nil {
		return fmt.Errorf("failed to update cmd/main.go: %w", err)
	}

	// go/v4's MainUpdater also runs during `create api` and injects
	// `mgr.GetClient()` / `mgr.GetScheme()` at the +kubebuilder:scaffold:builder
	// marker. mcmanager.Manager does not expose those methods directly; they must
	// be accessed via GetLocalManager(). Fix up any such occurrences here.
	if err := rewriteManagerCalls(s.fs); err != nil {
		return fmt.Errorf("failed to fix cmd/main.go: %w", err)
	}

	return nil
}

// rewriteManagerCalls replaces mgr.GetClient() and mgr.GetScheme() with their
// GetLocalManager() equivalents in cmd/main.go. go/v4's MainUpdater inserts
// these direct calls, but mcmanager.Manager does not implement them.
func rewriteManagerCalls(fs machinery.Filesystem) error {
	content, err := afero.ReadFile(fs.FS, "cmd/main.go")
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading cmd/main.go: %w", err)
	}

	fixed := strings.ReplaceAll(string(content), "mgr.GetClient()", "mgr.GetLocalManager().GetClient()")
	fixed = strings.ReplaceAll(fixed, "mgr.GetScheme()", "mgr.GetLocalManager().GetScheme()")

	if fixed == string(content) {
		return nil
	}

	return afero.WriteFile(fs.FS, "cmd/main.go", []byte(fixed), 0600)
}
