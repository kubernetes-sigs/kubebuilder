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

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/grafana/v1alpha/scaffolds/internal/templates"
)

var _ plugins.Scaffolder = &initScaffolder{}

type initScaffolder struct {
	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewInitScaffolder returns a new Scaffolder for project initialization operations
func NewInitScaffolder() plugins.Scaffolder {
	return &initScaffolder{}
}

// InjectFS implements cmdutil.Scaffolder
func (s *initScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *initScaffolder) Scaffold() error {
	log.Println("Generating Grafana manifests to visualize controller status...")

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs)

	err := scaffold.Execute(
		&templates.RuntimeManifest{},
		&templates.ResourcesManifest{},
		&templates.CustomMetricsConfigManifest{ConfigPath: configFilePath},
	)
	if err != nil {
		return fmt.Errorf("error scaffolding Grafana memanifests: %w", err)
	}

	return nil
}
