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
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/multi-module/v1alpha1/scaffolds/internal/templates/api"
	v3scaffolds "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds"
)

var _ plugins.Scaffolder = &apiScaffolder{}

const DefaultRequireVersion = "v0.0.0"

type apiScaffolder struct {
	config config.Config

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem

	multimodule  bool
	apiPath      string
	apiGoModPath string
	apiGoSumPath string
	apiModule    string
	dockerfile   string
}

// NewAPIScaffolder returns a new Scaffolder for  multi-module
func NewAPIScaffolder(config config.Config, apiPath string, multimodule bool) plugins.Scaffolder {
	return &apiScaffolder{
		config:       config,
		multimodule:  multimodule,
		apiPath:      apiPath,
		apiGoModPath: filepath.Join(apiPath, "go.mod"),
		apiGoSumPath: filepath.Join(apiPath, "go.sum"),
		apiModule:    config.GetRepository() + "/" + apiPath,
		dockerfile:   filepath.Join("Dockerfile"),
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *apiScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *apiScaffolder) Scaffold() error {
	if !s.multimodule {
		return s.cleanUp()
	}

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
	goMod := &api.GoMod{
		ControllerRuntimeVersion: v3scaffolds.ControllerRuntimeVersion,
		ModuleName:               s.apiModule,
	}
	goMod.Path = s.apiGoModPath
	if err = scaffold.Execute(goMod); err != nil {
		return fmt.Errorf("error updating scaffold: %w", err)
	}

	if err := util.RunCmd("require directive for new module in main module",
		"go", "mod", "edit", "-require",
		s.apiModule+"@"+DefaultRequireVersion); err != nil {
		return err
	}

	if err := util.RunCmd("adding replace directive for local folder in main module",
		"go", "mod", "edit", "-replace",
		s.apiModule+"="+"."+string(filepath.Separator)+s.apiPath); err != nil {
		return err
	}

	fmt.Println("updating Dockerfile to add module in the image")

	if err := util.InsertCode(s.dockerfile,
		"COPY go.sum go.sum",
		fmt.Sprintf("\n# Copy the Go Sub-Module manifests"+
			"\nCOPY %s %s"+
			"\nCOPY %s %s",
			s.apiGoModPath, s.apiGoModPath, s.apiGoSumPath, s.apiGoSumPath)); err != nil && err != util.ErrContentNotFound {
		return err
	}

	return nil
}

func (s *apiScaffolder) cleanUp() error {
	if err := s.fs.FS.Remove(s.apiGoModPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := s.fs.FS.Remove(filepath.Join(s.apiPath, "go.sum")); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := util.RunCmd("drop require directive", "go", "mod", "edit", "-droprequire",
		s.apiModule); err != nil {
		return err
	}

	if err := util.RunCmd("drop replace statement", "go", "mod", "edit", "-dropreplace",
		s.apiModule); err != nil {
		return err
	}

	fmt.Println("updating Dockerfile to remove module in the image")

	if err := util.ReplaceInFile(s.dockerfile, fmt.Sprintf("# Copy the Go Sub-Module manifests"+
		"\nCOPY %s %s"+
		"\nCOPY %s %s",
		s.apiGoModPath, s.apiGoModPath, s.apiGoSumPath, s.apiGoSumPath), ""); err != nil && err != util.ErrContentNotFound {
		return err
	}

	return nil
}
