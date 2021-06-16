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

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/config/certmanager"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/config/kdefault"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/config/manager"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/config/prometheus"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/config/rbac"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/config/webhook"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v2/scaffolds/internal/templates/hack"
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

var _ plugins.Scaffolder = &initScaffolder{}

type initScaffolder struct {
	config          config.Config
	boilerplatePath string
	license         string
	owner           string

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewInitScaffolder returns a new Scaffolder for project initialization operations
func NewInitScaffolder(config config.Config, license, owner string) plugins.Scaffolder {
	return &initScaffolder{
		config:          config,
		boilerplatePath: hack.DefaultBoilerplatePath,
		license:         license,
		owner:           owner,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *initScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *initScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")

	// Initialize the machinery.Scaffold that will write the boilerplate file to disk
	// The boilerplate file needs to be scaffolded as a separate step as it is going to
	// be used by the rest of the files, even those scaffolded in this command call.
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
	)

	bpFile := &hack.Boilerplate{
		License: s.license,
		Owner:   s.owner,
	}
	bpFile.Path = s.boilerplatePath
	if err := scaffold.Execute(bpFile); err != nil {
		return err
	}

	boilerplate, err := afero.ReadFile(s.fs.FS, s.boilerplatePath)
	if err != nil {
		return err
	}

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold = machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
		machinery.WithBoilerplate(string(boilerplate)),
	)

	return scaffold.Execute(
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
		&templates.Main{},
		&templates.GoMod{ControllerRuntimeVersion: ControllerRuntimeVersion},
		&templates.GitIgnore{},
		&templates.Makefile{
			Image:                  imageName,
			BoilerplatePath:        s.boilerplatePath,
			ControllerToolsVersion: ControllerToolsVersion,
			KustomizeVersion:       KustomizeVersion,
		},
		&templates.Dockerfile{},
		&kdefault.Kustomization{},
		&kdefault.ManagerAuthProxyPatch{},
		&kdefault.ManagerWebhookPatch{},
		&kdefault.WebhookCAInjectionPatch{},
		&webhook.Kustomization{},
		&webhook.KustomizeConfig{},
		&webhook.Service{},
		&prometheus.Kustomization{},
		&prometheus.Monitor{},
		&certmanager.Certificate{},
		&certmanager.Kustomization{},
		&certmanager.KustomizeConfig{},
	)
}
