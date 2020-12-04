/*
Copyright 2020 The Kubernetes Authors.

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
	"io/ioutil"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v2/pkg/model"
	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/config/certmanager"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/config/kdefault"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/config/manager"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/config/prometheus"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/config/rbac"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/hack"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/internal/machinery"
)

const (
	// ControllerRuntimeVersion is the kubernetes-sigs/controller-runtime version to be used in the project
	ControllerRuntimeVersion = "v0.7.0"
	// ControllerToolsVersion is the kubernetes-sigs/controller-tools version to be used in the project
	ControllerToolsVersion = "v0.4.1"
	// KustomizeVersion is the kubernetes-sigs/kustomize version to be used in the project
	KustomizeVersion = "v3.8.7"

	imageName = "controller:latest"
)

var _ cmdutil.Scaffolder = &initScaffolder{}

type initScaffolder struct {
	config          *config.Config
	boilerplatePath string
	license         string
	owner           string
}

// NewInitScaffolder returns a new Scaffolder for project initialization operations
func NewInitScaffolder(config *config.Config, license, owner string) cmdutil.Scaffolder {
	return &initScaffolder{
		config:          config,
		boilerplatePath: filepath.Join("hack", "boilerplate.go.txt"),
		license:         license,
		owner:           owner,
	}
}

func (s *initScaffolder) newUniverse(boilerplate string) *model.Universe {
	return model.NewUniverse(
		model.WithConfig(s.config),
		model.WithBoilerplate(boilerplate),
	)
}

// Scaffold implements Scaffolder
func (s *initScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")
	return s.scaffold()
}

// TODO: re-use universe created by s.newUniverse() if possible.
func (s *initScaffolder) scaffold() error {
	bpFile := &hack.Boilerplate{}
	bpFile.Path = s.boilerplatePath
	bpFile.License = s.license
	bpFile.Owner = s.owner
	if err := machinery.NewScaffold().Execute(
		s.newUniverse(""),
		bpFile,
	); err != nil {
		return err
	}

	boilerplate, err := ioutil.ReadFile(s.boilerplatePath) //nolint:gosec
	if err != nil {
		return err
	}

	return machinery.NewScaffold().Execute(
		s.newUniverse(string(boilerplate)),
		&rbac.Kustomization{},
		&rbac.AuthProxyRole{},
		&rbac.AuthProxyRoleBinding{},
		&rbac.AuthProxyService{},
		&rbac.AuthProxyClientRole{},
		&rbac.RoleBinding{},
		&rbac.LeaderElectionRole{},
		&rbac.LeaderElectionRoleBinding{},
		&manager.Kustomization{},
		&manager.Config{Image: imageName},
		&manager.ControllerManagerConfig{},
		&templates.Main{},
		&templates.GoMod{ControllerRuntimeVersion: ControllerRuntimeVersion},
		&templates.GitIgnore{},
		&templates.Makefile{
			Image:                    imageName,
			BoilerplatePath:          s.boilerplatePath,
			ControllerToolsVersion:   ControllerToolsVersion,
			KustomizeVersion:         KustomizeVersion,
			ControllerRuntimeVersion: ControllerRuntimeVersion,
		},
		&templates.Dockerfile{},
		&templates.DockerIgnore{},
		&kdefault.Kustomization{},
		&kdefault.ManagerAuthProxyPatch{},
		&kdefault.ManagerConfigPatch{},
		&prometheus.Kustomization{},
		&prometheus.Monitor{},
		&certmanager.Certificate{},
		&certmanager.Kustomization{},
		&certmanager.KustomizeConfig{},
	)
}
