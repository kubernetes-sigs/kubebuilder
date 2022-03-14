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

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/config-gen/v1alpha/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/config-gen/v1alpha/scaffolds/internal/templates/config/configgen"
)

const (
	// ControllerRuntimeVersion is the kubernetes-sigs/controller-runtime version to be used in the project
	ControllerRuntimeVersion = "v0.11.0"
	// ControllerToolsVersion is the kubernetes-sigs/controller-tools version to be used in the project
	ControllerToolsVersion = "v0.8.0"
	// KustomizeVersion is the kubernetes-sigs/kustomize version to be used in the project
	KustomizeVersion = "v4.0.5"
)

var _ plugins.Scaffolder = &initScaffolder{}

type initScaffolder struct {
	config config.Config

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem

	boilerplatePath string
}

// NewInitScaffolder returns a new Scaffolder for project initialization operations
func NewInitScaffolder(config config.Config, bpPath string) plugins.Scaffolder {
	return &initScaffolder{
		config:          config,
		boilerplatePath: bpPath,
	}
}

func (s *initScaffolder) InjectFS(fs machinery.Filesystem) { s.fs = fs }

func (s *initScaffolder) Scaffold() error {
	fmt.Println("Writing config-gen manifests for you to edit...")

	scaffold := machinery.NewScaffold(s.fs, machinery.WithConfig(s.config))

	configGen := configgen.ConfigGen{}
	cmConfig := configgen.ControllerManagerConfig{}

	builders := []machinery.Builder{
		&configGen,
		&cmConfig,
		&templates.Makefile{
			BoilerplatePath:          s.boilerplatePath,
			ControllerToolsVersion:   ControllerToolsVersion,
			KustomizeVersion:         KustomizeVersion,
			ControllerRuntimeVersion: ControllerRuntimeVersion,
		},
	}

	if err := scaffold.Execute(builders...); err != nil {
		return err
	}

	return nil
}
