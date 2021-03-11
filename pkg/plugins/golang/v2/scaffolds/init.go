/*
Copyright 2019 The Kubernetes Authors.

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
	"sigs.k8s.io/kubebuilder/v3/pkg/model"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/config/certmanager"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/config/kdefault"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/config/manager"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/config/prometheus"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/config/rbac"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/config/webhook"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/hack"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/internal/machinery"
)

const (
	// ControllerRuntimeVersion is the kubernetes-sigs/controller-runtime version to be used in the project
	ControllerRuntimeVersion = "v0.6.4"
	// ControllerToolsVersion is the kubernetes-sigs/controller-tools version to be used in the project
	ControllerToolsVersion = "v0.3.0"
	// KustomizeVersion is the kubernetes-sigs/kustomize version to be used in the project
	KustomizeVersion = "v3.5.4"

	imageName = "controller:latest"
)

var _ cmdutil.Scaffolder = &initScaffolder{}

type initScaffolder struct {
	config          config.Config
	boilerplatePath string
	license         string
	owner           string

	// fs is the filesystem that will be used by the scaffolder
	fs afero.Fs
}

// NewInitScaffolder returns a new Scaffolder for project initialization operations
func NewInitScaffolder(config config.Config, license, owner string) cmdutil.Scaffolder {
	return &initScaffolder{
		config:          config,
		boilerplatePath: filepath.Join("hack", "boilerplate.go.txt"),
		license:         license,
		owner:           owner,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *initScaffolder) InjectFS(fs afero.Fs) {
	s.fs = fs
}

func (s *initScaffolder) newUniverse(boilerplate string) *model.Universe {
	return model.NewUniverse(
		model.WithConfig(s.config),
		model.WithBoilerplate(boilerplate),
	)
}

// Scaffold implements cmdutil.Scaffolder
func (s *initScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")

	bpFile := &hack.Boilerplate{}
	bpFile.Path = s.boilerplatePath
	bpFile.License = s.license
	bpFile.Owner = s.owner
	if err := machinery.NewScaffold(s.fs).Execute(
		s.newUniverse(""),
		bpFile,
	); err != nil {
		return err
	}

	boilerplate, err := afero.ReadFile(s.fs, s.boilerplatePath)
	if err != nil {
		return err
	}

	return machinery.NewScaffold(s.fs).Execute(
		s.newUniverse(string(boilerplate)),
		&templates.GitIgnore{},
		&rbac.AuthProxyRole{},
		&rbac.AuthProxyRoleBinding{},
		&kdefault.ManagerAuthProxyPatch{},
		&rbac.AuthProxyService{},
		&rbac.AuthProxyClientRole{},
		&manager.Config{Image: imageName},
		&templates.Main{},
		&templates.GoMod{ControllerRuntimeVersion: ControllerRuntimeVersion},
		&templates.Makefile{
			Image:                  imageName,
			BoilerplatePath:        s.boilerplatePath,
			ControllerToolsVersion: ControllerToolsVersion,
			KustomizeVersion:       KustomizeVersion,
		},
		&templates.Dockerfile{},
		&kdefault.Kustomization{},
		&kdefault.ManagerWebhookPatch{},
		&rbac.RoleBinding{},
		&rbac.LeaderElectionRole{},
		&rbac.LeaderElectionRoleBinding{},
		&rbac.Kustomization{},
		&manager.Kustomization{},
		&webhook.Kustomization{},
		&webhook.KustomizeConfig{},
		&webhook.Service{},
		&kdefault.WebhookCAInjectionPatch{},
		&prometheus.Kustomization{},
		&prometheus.Monitor{},
		&certmanager.Certificate{},
		&certmanager.Kustomization{},
		&certmanager.KustomizeConfig{},
	)
}
