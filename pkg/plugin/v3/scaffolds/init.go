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

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin/internal/machinery"
	"sigs.k8s.io/kubebuilder/pkg/plugin/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/plugin/v3/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/pkg/plugin/v3/scaffolds/internal/templates/config/certmanager"
	"sigs.k8s.io/kubebuilder/pkg/plugin/v3/scaffolds/internal/templates/config/hack"
	"sigs.k8s.io/kubebuilder/pkg/plugin/v3/scaffolds/internal/templates/config/kdefault"
	"sigs.k8s.io/kubebuilder/pkg/plugin/v3/scaffolds/internal/templates/config/manager"
	"sigs.k8s.io/kubebuilder/pkg/plugin/v3/scaffolds/internal/templates/config/prometheus"
	"sigs.k8s.io/kubebuilder/pkg/plugin/v3/scaffolds/internal/templates/config/rbac"
	"sigs.k8s.io/kubebuilder/pkg/plugin/v3/scaffolds/internal/templates/config/webhook"
)

const (
	// ControllerRuntimeVersion is the kubernetes-sigs/controller-runtime version to be used in the project
	ControllerRuntimeVersion = "v0.6.3"
	// ControllerToolsVersion is the kubernetes-sigs/controller-tools version to be used in the project
	ControllerToolsVersion = "v0.3.0"
	// KustomizeVersion is the kubernetes-sigs/kustomize version to be used in the project
	KustomizeVersion = "v3.5.4"

	imageName = "controller:latest"
)

var _ scaffold.Scaffolder = &initScaffolder{}

type initScaffolder struct {
	config          *config.Config
	boilerplatePath string
	license         string
	owner           string
}

// NewInitScaffolder returns a new Scaffolder for project initialization operations
func NewInitScaffolder(config *config.Config, license, owner string) scaffold.Scaffolder {
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
		&templates.GitIgnore{},
		&rbac.AuthProxyRole{},
		&rbac.AuthProxyRoleBinding{},
		&kdefault.AuthProxyPatch{},
		&rbac.AuthProxyService{},
		&rbac.ClientClusterRole{},
		&manager.Config{Image: imageName},
		&templates.Main{},
		&templates.GoMod{ControllerRuntimeVersion: ControllerRuntimeVersion},
		&templates.Makefile{
			Image:                    imageName,
			BoilerplatePath:          s.boilerplatePath,
			ControllerToolsVersion:   ControllerToolsVersion,
			KustomizeVersion:         KustomizeVersion,
			ControllerRuntimeVersion: ControllerRuntimeVersion,
		},
		&templates.Dockerfile{},
		&templates.DockerignoreFile{},
		&kdefault.Kustomize{},
		&kdefault.ManagerWebhookPatch{},
		&rbac.ManagerRoleBinding{},
		&rbac.LeaderElectionRole{},
		&rbac.LeaderElectionRoleBinding{},
		&rbac.KustomizeRBAC{},
		&manager.Kustomization{},
		&webhook.Kustomization{},
		&webhook.KustomizeConfigWebhook{},
		&webhook.Service{},
		&kdefault.InjectCAPatch{},
		&prometheus.Kustomization{},
		&prometheus.ServiceMonitor{},
		&certmanager.CertManager{},
		&certmanager.Kustomization{},
		&certmanager.KustomizeConfig{},
	)
}
