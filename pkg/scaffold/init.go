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

package scaffold

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/internal/machinery"
	templatesv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v1"
	managerv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v1/manager"
	metricsauthv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v1/metricsauth"
	templatesv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v2"
	certmanagerv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v2/certmanager"
	managerv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v2/manager"
	metricsauthv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v2/metricsauth"
	prometheusv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v2/prometheus"
	webhookv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v2/webhook"
)

const (
	// ControllerRuntimeVersion is the kubernetes-sigs/controller-runtime version to be used in the project
	ControllerRuntimeVersion = "v0.6.0"
	// ControllerToolsVersion is the kubernetes-sigs/controller-tools version to be used in the project
	ControllerToolsVersion = "v0.2.5"
	// KustomizeVersion is the kubernetes-sigs/kustomize version to be used in the project
	KustomizeVersion = "v3.5.4"

	imageName = "controller:latest"
)

var _ Scaffolder = &initScaffolder{}

type initScaffolder struct {
	config          *config.Config
	boilerplatePath string
	license         string
	owner           string
}

// NewInitScaffolder returns a new Scaffolder for project initialization operations
func NewInitScaffolder(config *config.Config, license, owner string) Scaffolder {
	return &initScaffolder{
		config:          config,
		boilerplatePath: filepath.Join("hack", "boilerplate.go.txt"),
		license:         license,
		owner:           owner,
	}
}

func (s *initScaffolder) newUniverse(boilerplate string) *model.Universe {
	return model.NewUniverse(
		model.WithConfig(&s.config.Config),
		model.WithBoilerplate(boilerplate),
	)
}

// Scaffold implements Scaffolder
func (s *initScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")

	if err := s.config.Save(); err != nil {
		return err
	}

	switch {
	case s.config.IsV1():
		return s.scaffoldV1()
	case s.config.IsV2():
		return s.scaffoldV2()
	default:
		return fmt.Errorf("unknown project version %v", s.config.Version)
	}
}

func (s *initScaffolder) scaffoldV1() error {
	bpFile := &templatesv1.Boilerplate{}
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
		&templatesv1.GitIgnore{},
		&templatesv1.AuthProxyRole{},
		&templatesv1.AuthProxyRoleBinding{},
		&templatesv1.KustomizeRBAC{},
		&templatesv1.KustomizeImagePatch{},
		&metricsauthv1.KustomizePrometheusMetricsPatch{},
		&metricsauthv1.KustomizeAuthProxyPatch{},
		&templatesv1.AuthProxyService{},
		&managerv1.Config{Image: imageName},
		&templatesv1.Makefile{Image: imageName},
		&templatesv1.GopkgToml{},
		&managerv1.Dockerfile{},
		&templatesv1.Kustomize{},
		&templatesv1.KustomizeManager{},
		&managerv1.APIs{BoilerplatePath: s.boilerplatePath},
		&managerv1.Controller{},
		&managerv1.Webhook{},
		&managerv1.Cmd{},
	)
}

func (s *initScaffolder) scaffoldV2() error {
	bpFile := &templatesv2.Boilerplate{}
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
		&templatesv2.GitIgnore{},
		&templatesv2.AuthProxyRole{},
		&templatesv2.AuthProxyRoleBinding{},
		&metricsauthv2.AuthProxyPatch{},
		&metricsauthv2.AuthProxyService{},
		&metricsauthv2.ClientClusterRole{},
		&managerv2.Config{Image: imageName},
		&templatesv2.Main{},
		&templatesv2.GoMod{ControllerRuntimeVersion: ControllerRuntimeVersion},
		&templatesv2.Makefile{
			Image:                  imageName,
			BoilerplatePath:        s.boilerplatePath,
			ControllerToolsVersion: ControllerToolsVersion,
			KustomizeVersion:       KustomizeVersion,
		},
		&templatesv2.Dockerfile{},
		&templatesv2.Kustomize{},
		&templatesv2.ManagerWebhookPatch{},
		&templatesv2.ManagerRoleBinding{},
		&templatesv2.LeaderElectionRole{},
		&templatesv2.LeaderElectionRoleBinding{},
		&templatesv2.KustomizeRBAC{},
		&managerv2.Kustomization{},
		&webhookv2.Kustomization{},
		&webhookv2.KustomizeConfigWebhook{},
		&webhookv2.Service{},
		&webhookv2.InjectCAPatch{},
		&prometheusv2.Kustomization{},
		&prometheusv2.ServiceMonitor{},
		&certmanagerv2.CertManager{},
		&certmanagerv2.Kustomization{},
		&certmanagerv2.KustomizeConfig{},
	)
}
