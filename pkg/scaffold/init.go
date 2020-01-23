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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/project"
	scaffoldv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/v1"
	managerv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/v1/manager"
	metricsauthv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/v1/metricsauth"
	scaffoldv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2"
	certmanagerv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2/certmanager"
	managerv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2/manager"
	metricsauthv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2/metricsauth"
	prometheusv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2/prometheus"
	webhookv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2/webhook"
)

const (
	// controller runtime version to be used in the project
	ControllerRuntimeVersion = "v0.4.0"
	// ControllerTools version to be used in the project
	ControllerToolsVersion = "v0.2.4"

	ImageName = "controller:latest"
)

type initScaffolder struct {
	config          *config.Config
	boilerplatePath string
	license         string
	owner           string
}

func NewInitScaffolder(config *config.Config, license, owner string) Scaffolder {
	return &initScaffolder{
		config:          config,
		boilerplatePath: filepath.Join("hack", "boilerplate.go.txt"),
		license:         license,
		owner:           owner,
	}
}

func (s *initScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")

	if err := s.config.Save(); err != nil {
		return err
	}

	universe, err := model.NewUniverse(
		model.WithConfig(&s.config.Config),
		model.WithoutBoilerplate,
	)
	if err != nil {
		return fmt.Errorf("error initializing project: %v", err)
	}

	if err := (&Scaffold{BoilerplateOptional: true}).Execute(
		universe,
		input.Options{ProjectPath: s.config.Path(), BoilerplatePath: s.boilerplatePath},
		&project.Boilerplate{
			Input:   input.Input{Path: s.boilerplatePath},
			License: s.license,
			Owner:   s.owner,
		},
	); err != nil {
		return err
	}

	universe, err = model.NewUniverse(
		model.WithConfig(&s.config.Config),
		model.WithBoilerplateFrom(s.boilerplatePath),
	)
	if err != nil {
		return fmt.Errorf("error initializing project: %v", err)
	}

	if err := (&Scaffold{}).Execute(
		universe,
		input.Options{ProjectPath: s.config.Path(), BoilerplatePath: s.boilerplatePath},
		&project.GitIgnore{},
		&project.AuthProxyRole{},
		&project.AuthProxyRoleBinding{},
	); err != nil {
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
	universe, err := model.NewUniverse(
		model.WithConfig(&s.config.Config),
		model.WithBoilerplateFrom(s.boilerplatePath),
	)
	if err != nil {
		return fmt.Errorf("error initializing project: %v", err)
	}

	return (&Scaffold{}).Execute(
		universe,
		input.Options{ProjectPath: s.config.Path(), BoilerplatePath: s.boilerplatePath},
		&project.KustomizeRBAC{},
		&scaffoldv1.KustomizeImagePatch{},
		&metricsauthv1.KustomizePrometheusMetricsPatch{},
		&metricsauthv1.KustomizeAuthProxyPatch{},
		&scaffoldv1.AuthProxyService{},
		&managerv1.Config{Image: ImageName},
		&project.Makefile{Image: ImageName},
		&project.GopkgToml{},
		&managerv1.Dockerfile{},
		&project.Kustomize{},
		&project.KustomizeManager{},
		&managerv1.APIs{},
		&managerv1.Controller{},
		&managerv1.Webhook{},
		&managerv1.Cmd{},
	)
}

func (s *initScaffolder) scaffoldV2() error {
	universe, err := model.NewUniverse(
		model.WithConfig(&s.config.Config),
		model.WithBoilerplateFrom(s.boilerplatePath),
	)
	if err != nil {
		return fmt.Errorf("error initializing project: %v", err)
	}

	return (&Scaffold{}).Execute(
		universe,
		input.Options{ProjectPath: s.config.Path(), BoilerplatePath: s.boilerplatePath},
		&metricsauthv2.AuthProxyPatch{},
		&metricsauthv2.AuthProxyService{},
		&metricsauthv2.ClientClusterRole{},
		&managerv2.Config{Image: ImageName},
		&scaffoldv2.Main{},
		&scaffoldv2.GoMod{ControllerRuntimeVersion: ControllerRuntimeVersion},
		&scaffoldv2.Makefile{Image: ImageName, ControllerToolsVersion: ControllerToolsVersion},
		&scaffoldv2.Dockerfile{},
		&scaffoldv2.Kustomize{},
		&scaffoldv2.ManagerWebhookPatch{},
		&scaffoldv2.ManagerRoleBinding{},
		&scaffoldv2.LeaderElectionRole{},
		&scaffoldv2.LeaderElectionRoleBinding{},
		&scaffoldv2.KustomizeRBAC{},
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
