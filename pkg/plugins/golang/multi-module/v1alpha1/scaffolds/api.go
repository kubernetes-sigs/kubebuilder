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
	"path/filepath"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/multi-module/v1alpha1/scaffolds/internal/templates/api"
	v3scaffolds "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds"
)

var _ plugins.Scaffolder = &apiScaffolder{}

type apiScaffolder struct {
	config config.Config

	goModPath string

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewAPIScaffolder returns a new Scaffolder for  multi-module
func NewAPIScaffolder(config config.Config, goModPath string) plugins.Scaffolder {
	return &apiScaffolder{
		config:    config,
		goModPath: goModPath,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *apiScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *apiScaffolder) Scaffold() error {
	// Load the boilerplate
	boilerplate, err := afero.ReadFile(s.fs.FS, filepath.Join("hack", "boilerplate.go.txt"))
	if err != nil {
		return fmt.Errorf("error updating scaffold: unable to load boilerplate: %w", err)
	}

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
		machinery.WithBoilerplate(string(boilerplate)),
	)

	err = scaffold.Execute(
		&api.GoMod{
			ControllerRuntimeVersion: v3scaffolds.ControllerRuntimeVersion,
			Path:                     s.goModPath,
		},
	)
	if err != nil {
		return fmt.Errorf("error updating scaffold: %w", err)
	}

	return nil
}
