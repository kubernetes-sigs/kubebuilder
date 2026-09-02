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

const (
	// MulticlusterRuntimeVersion is the version of sigs.k8s.io/multicluster-runtime
	// to be used in the scaffolded project.
	MulticlusterRuntimeVersion = "v0.23.3"

	// ClusterAPIVersion is the version of sigs.k8s.io/cluster-api required by the
	// cluster-api provider at MulticlusterRuntimeVersion. It must be kept in sync with
	// the `require` directive in providers/cluster-api/go.mod at that version tag.
	// We pin it explicitly because go/v4's `go mod tidy` may resolve cluster-api to a
	// newer, incompatible version before our PostScaffold runs.
	ClusterAPIVersion = "v1.9.4"
)

var _ plugins.Scaffolder = &initScaffolder{}

type initScaffolder struct {
	config        config.Config
	provider      string
	kubeconfigDir string
	fs            machinery.Filesystem
}

// NewInitScaffolder returns a Scaffolder for the init command.
func NewInitScaffolder(cfg config.Config, provider, kubeconfigDir string) plugins.Scaffolder {
	return &initScaffolder{
		config:        cfg,
		provider:      provider,
		kubeconfigDir: kubeconfigDir,
	}
}

// InjectFS implements plugins.Scaffolder.
func (s *initScaffolder) InjectFS(fs machinery.Filesystem) { s.fs = fs }

// Scaffold writes the multicluster-aware cmd/main.go, overwriting go/v4's version.
func (s *initScaffolder) Scaffold() error {
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
