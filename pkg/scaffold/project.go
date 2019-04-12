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

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/project"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/manager"

	scaffoldv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2"
)

// Project contains configuration for generating project scaffolding.
type Project struct {
	scaffold *Scaffold

	Info        project.Project
	Boilerplate project.Boilerplate
}

func (p *Project) Scaffold() error {
	// project and boilerplate must come before main so the boilerplate exists
	s := &Scaffold{
		BoilerplateOptional: true,
		ProjectOptional:     true,
	}

	projectInput, err := p.Info.GetInput()
	if err != nil {
		return err
	}

	bpInput, err := p.Boilerplate.GetInput()
	if err != nil {
		return err
	}

	err = s.Execute(
		input.Options{ProjectPath: projectInput.Path, BoilerplatePath: bpInput.Path},
		&p.Info,
		&p.Boilerplate,
	)
	if err != nil {
		return err
	}

	// default controller manager image name
	imgName := "controller:latest"

	s = &Scaffold{}
	err = s.Execute(
		input.Options{ProjectPath: projectInput.Path, BoilerplatePath: bpInput.Path},
		&manager.Config{Image: imgName},
		&project.GitIgnore{},
		&project.Kustomize{},
		&project.KustomizeRBAC{},
		&project.KustomizeManager{},
		&project.KustomizeImagePatch{},
		&project.KustomizePrometheusMetricsPatch{},
		&project.KustomizeAuthProxyPatch{},
		&project.AuthProxyService{},
		&project.AuthProxyRole{},
		&project.AuthProxyRoleBinding{})
	if err != nil {
		return err
	}

	switch ver := projectInput.Version; ver {
	case project.Version1:
		return p.scaffoldV1()
	case project.Version2:
		return p.scaffoldV2()
	default:
		return fmt.Errorf("unknown project version '%v'", ver)
	}
	return nil
}

func (p *Project) setDefaults() error {
	return nil
}

func (p *Project) Validate() error {
	return nil
}

func (p *Project) scaffoldV1() error {
	// default controller manager image name
	imgName := "controller:latest"
	return (&Scaffold{}).Execute(
		input.Options{ProjectPath: p.Info.Path, BoilerplatePath: p.Boilerplate.Path},
		&project.Makefile{Image: imgName},
		&project.GopkgToml{},
		&manager.Dockerfile{},
		&manager.APIs{},
		&manager.Controller{},
		&manager.Webhook{},
		&manager.Cmd{},
	)
}

func (p *Project) scaffoldV2() error {
	// default controller manager image name
	imgName := "controller:latest"
	return (&Scaffold{}).Execute(
		input.Options{ProjectPath: p.Info.Path, BoilerplatePath: p.Boilerplate.Path},
		&scaffoldv2.Main{},
		&scaffoldv2.GopkgToml{},
		&scaffoldv2.Doc{},
		&scaffoldv2.Makefile{Image: imgName},
		&scaffoldv2.Dockerfile{},
	)
}
