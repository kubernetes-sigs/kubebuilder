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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"sigs.k8s.io/kubebuilder/cmd/util"
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/project"
	scaffoldv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/v1"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/manager"
	metricsauthv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/v1/metricsauth"
	scaffoldv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v2/certmanager"
	managerv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2/manager"
	metricsauthv2 "sigs.k8s.io/kubebuilder/pkg/scaffold/v2/metricsauth"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v2/webhook"
)

const (
	// controller runtime version to be used in the project
	controllerRuntimeVersion = "v0.2.2"
	// ControllerTools version to be used in the project
	controllerToolsVersion = "v0.2.1"
)

type ProjectScaffolder interface {
	EnsureDependencies() (bool, error)
	Scaffold() error
	Validate() error
}

type V1Project struct {
	Project     project.Project
	Boilerplate project.Boilerplate

	DepArgs          []string
	DefinitelyEnsure *bool
}

func (p *V1Project) Validate() error {
	_, err := exec.LookPath("dep")
	if err != nil {
		return fmt.Errorf("dep is not installed (%v). Follow steps at: https://golang.github.io/dep/docs/installation.html", err)
	}
	return nil
}

func (p *V1Project) EnsureDependencies() (bool, error) {
	if p.DefinitelyEnsure == nil {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Run `dep ensure` to fetch dependencies (Recommended) [y/n]?")
		if !util.Yesno(reader) {
			return false, nil
		}
	} else if !*p.DefinitelyEnsure {
		return false, nil
	}

	c := exec.Command("dep", "ensure") // #nosec
	c.Args = append(c.Args, p.DepArgs...)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	fmt.Println(strings.Join(c.Args, " "))
	return true, c.Run()
}

func (p *V1Project) buildUniverse() *model.Universe {
	return &model.Universe{}
}

func (p *V1Project) Scaffold() error {
	p.Project.Version = project.Version1

	s := &Scaffold{
		BoilerplateOptional: true,
		ProjectOptional:     true,
	}

	projectInput, err := p.Project.GetInput()
	if err != nil {
		return err
	}

	bpInput, err := p.Boilerplate.GetInput()
	if err != nil {
		return err
	}

	err = s.Execute(
		p.buildUniverse(),
		input.Options{ProjectPath: projectInput.Path, BoilerplatePath: bpInput.Path},
		&p.Project,
		&p.Boilerplate,
	)
	if err != nil {
		return err
	}

	// default controller manager image name
	imgName := "controller:latest"

	s = &Scaffold{}
	return s.Execute(
		p.buildUniverse(),
		input.Options{ProjectPath: projectInput.Path, BoilerplatePath: bpInput.Path},
		&project.GitIgnore{},
		&project.KustomizeRBAC{},
		&scaffoldv1.KustomizeImagePatch{},
		&metricsauthv1.KustomizePrometheusMetricsPatch{},
		&metricsauthv1.KustomizeAuthProxyPatch{},
		&scaffoldv1.AuthProxyService{},
		&project.AuthProxyRole{},
		&project.AuthProxyRoleBinding{},
		&manager.Config{Image: imgName},
		&project.Makefile{Image: imgName},
		&project.GopkgToml{},
		&manager.Dockerfile{},
		&project.Kustomize{},
		&project.KustomizeManager{},
		&manager.APIs{},
		&manager.Controller{},
		&manager.Webhook{},
		&manager.Cmd{})
}

type V2Project struct {
	Project     project.Project
	Boilerplate project.Boilerplate
}

func (p *V2Project) Validate() error {
	return nil
}

func (p *V2Project) EnsureDependencies() (bool, error) {
	// ensure that we are pinning controller-runtime version
	// xref: https://github.com/kubernetes-sigs/kubebuilder/issues/997
	c := exec.Command("go", "get", "sigs.k8s.io/controller-runtime@"+controllerRuntimeVersion) // #nosec
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	fmt.Println(strings.Join(c.Args, " "))
	err := c.Run()
	if err != nil {
		return false, err
	}

	c = exec.Command("go", "mod", "tidy") // #nosec
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	fmt.Println(strings.Join(c.Args, " "))
	err = c.Run()
	if err != nil {
		return false, err
	}
	return true, err
}

func (p *V2Project) buildUniverse() *model.Universe {
	return &model.Universe{}
}

func (p *V2Project) Scaffold() error {
	p.Project.Version = project.Version2

	s := &Scaffold{
		BoilerplateOptional: true,
		ProjectOptional:     true,
	}

	projectInput, err := p.Project.GetInput()
	if err != nil {
		return err
	}

	bpInput, err := p.Boilerplate.GetInput()
	if err != nil {
		return err
	}

	err = s.Execute(
		p.buildUniverse(),
		input.Options{ProjectPath: projectInput.Path, BoilerplatePath: bpInput.Path},
		&p.Project,
		&p.Boilerplate,
	)
	if err != nil {
		return err
	}

	// default controller manager image name
	imgName := "controller:latest"

	s = &Scaffold{}
	return s.Execute(
		p.buildUniverse(),
		input.Options{ProjectPath: projectInput.Path, BoilerplatePath: bpInput.Path},
		&project.GitIgnore{},
		&metricsauthv2.KustomizePrometheusMetricsPatch{},
		&metricsauthv2.KustomizeAuthProxyPatch{},
		&scaffoldv2.AuthProxyService{},
		&project.AuthProxyRole{},
		&project.AuthProxyRoleBinding{},
		&managerv2.Config{Image: imgName},
		&scaffoldv2.Main{},
		&scaffoldv2.GoMod{ControllerRuntimeVersion: controllerRuntimeVersion},
		&scaffoldv2.Makefile{Image: imgName, ControllerToolsVersion: controllerToolsVersion},
		&scaffoldv2.Dockerfile{},
		&scaffoldv2.Kustomize{},
		&scaffoldv2.ManagerWebhookPatch{},
		&scaffoldv2.ManagerRoleBinding{},
		&scaffoldv2.LeaderElectionRole{},
		&scaffoldv2.LeaderElectionRoleBinding{},
		&scaffoldv2.KustomizeRBAC{},
		&managerv2.Kustomization{},
		&webhook.Kustomization{},
		&webhook.KustomizeConfigWebhook{},
		&webhook.Service{},
		&webhook.InjectCAPatch{},
		&certmanager.CertManager{},
		&certmanager.Kustomization{},
		&certmanager.KustomizeConfig{})
}
