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

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/multicluster-runtime/v1alpha/scaffolds/internal/templates/cmd"
)

var _ plugins.Scaffolder = &editScaffolder{}

type editScaffolder struct {
	config        config.Config
	provider      string
	kubeconfigDir string
	fs            machinery.Filesystem
}

// NewEditScaffolder returns a Scaffolder for the edit command.
func NewEditScaffolder(cfg config.Config, provider, kubeconfigDir string) plugins.Scaffolder {
	return &editScaffolder{
		config:        cfg,
		provider:      provider,
		kubeconfigDir: kubeconfigDir,
	}
}

// InjectFS implements plugins.Scaffolder.
func (s *editScaffolder) InjectFS(fs machinery.Filesystem) { s.fs = fs }

// Scaffold rewrites cmd/main.go to switch to the selected provider.
func (s *editScaffolder) Scaffold() error {
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
	)

	if err := scaffold.Execute(
		&cmd.Main{
			Provider:                   s.provider,
			KubeconfigDir:              s.kubeconfigDir,
			MulticlusterRuntimeVersion: MulticlusterRuntimeVersion,
		},
	); err != nil {
		return fmt.Errorf("failed to execute scaffold: %w", err)
	}
	return nil
}
