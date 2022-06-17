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
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds/internal/templates/api"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds/internal/templates/config/samples"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds/internal/templates/controllers"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds"
)

var _ plugins.Scaffolder = &apiScaffolder{}

type apiScaffolder struct {
	config   config.Config
	resource resource.Resource
	image    string

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewAPIScaffolder returns a new Scaffolder for declarative
func NewAPIScaffolder(config config.Config, res resource.Resource, image string) plugins.Scaffolder {
	return &apiScaffolder{
		config:   config,
		resource: res,
		image:    image,
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
		machinery.WithResource(&s.resource),
	)

	err = scaffold.Execute(
		&controllers.Controller{ControllerRuntimeVersion: scaffolds.ControllerRuntimeVersion, Image: s.image},
		&api.Types{},
		&samples.CRDSample{},
	)
	if err != nil {
		return fmt.Errorf("error updating scaffold: %w", err)
	}
	return nil
}
