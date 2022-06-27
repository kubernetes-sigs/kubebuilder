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
	"strings"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
	kustomizev1scaffolds "sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v1/scaffolds"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds/internal/templates/api"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds/internal/templates/config/samples"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds/internal/templates/controllers"
	golangv3scaffolds "sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds"
)

var _ plugins.Scaffolder = &apiScaffolder{}

// apiScaffolder contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type apiScaffolder struct {
	config   config.Config
	resource resource.Resource
	image    string
	command  string
	port     string

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewAPIScaffolder returns a new Scaffolder for declarative
//nolint: lll
func NewDeployImageScaffolder(config config.Config, res resource.Resource, image,
	command, port string) plugins.Scaffolder {
	return &apiScaffolder{
		config:   config,
		resource: res,
		image:    image,
		command:  command,
		port:     port,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *apiScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *apiScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")

	if err := s.scaffoldCreateAPIFromGolang(); err != nil {
		return fmt.Errorf("error scaffolding APIs: %v", err)
	}

	if err := s.scaffoldCreateAPIFromKustomize(); err != nil {
		return fmt.Errorf("error scaffolding kustomize file for the new APIs: %v", err)
	}

	// Load the boilerplate
	boilerplate, err := afero.ReadFile(s.fs.FS, filepath.Join("hack", "boilerplate.go.txt"))
	if err != nil {
		return fmt.Errorf("error scaffolding API/controller: unable to load boilerplate: %w", err)
	}

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
		machinery.WithBoilerplate(string(boilerplate)),
		machinery.WithResource(&s.resource),
	)

	if err := scaffold.Execute(
		&api.Types{Port: s.port},
	); err != nil {
		return fmt.Errorf("error updating APIs: %v", err)
	}

	if err := s.scafffoldControllerWithImage(scaffold); err != nil {
		return fmt.Errorf("error updating controller: %v", err)
	}

	if err := scaffold.Execute(
		&samples.CRDSample{Port: s.port},
	); err != nil {
		return fmt.Errorf("error updating config/samples: %v", err)
	}

	return nil
}

func (s *apiScaffolder) scafffoldControllerWithImage(scaffold *machinery.Scaffold) error {
	controller := &controllers.Controller{ControllerRuntimeVersion: golangv3scaffolds.ControllerRuntimeVersion,
		Image: s.image,
	}
	if err := scaffold.Execute(
		controller,
	); err != nil {
		return fmt.Errorf("error scaffolding controller: %v", err)
	}

	controllerPath := controller.Path
	if err := util.ReplaceInFile(controllerPath, "//TODO: scaffold container",
		fmt.Sprintf(containerTemplate,
			s.image,                          // value for the image
			strings.ToLower(s.resource.Kind), // value for the name of the container
		),
	); err != nil {
		return fmt.Errorf("error scaffolding container in the controller: %v", err)
	}

	// Scaffold the command if informed
	if len(s.command) > 0 {
		// TODO: improve it to be an spec in the sample and api instead so that
		// users can change the values
		var res string
		for _, value := range strings.Split(s.command, ",") {
			res += fmt.Sprintf("\"%s\",", strings.TrimSpace(value))
		}
		res = res[:len(res)-1]
		err := util.InsertCode(controllerPath, `SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &[]bool{true}[0],
							RunAsUser: &[]int64{1000}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},`, fmt.Sprintf(commandTemplate, res))
		if err != nil {
			return fmt.Errorf("error scaffolding command in the controller: %v", err)
		}
	}

	// Scaffold the port if informed
	if len(s.port) > 0 {
		err := util.InsertCode(controllerPath, `SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &[]bool{true}[0],
							RunAsUser: &[]int64{1000}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},`, fmt.Sprintf(portTemplate, strings.ToLower(s.resource.Kind)))
		if err != nil {
			return fmt.Errorf("error scaffolding container port in the controller: %v", err)
		}
	}
	return nil
}

func (s *apiScaffolder) scaffoldCreateAPIFromKustomize() error {
	// Now we need call the kustomize/v1 plugin to do its scaffolds when we create a new API
	// todo: when we have the go/v4-alpha plugin we will also need to check what is the plugin used
	// in the Project layout to know if we should use kustomize/v1 OR kustomize/v2-alpha
	kustomizeV1Scaffolder := kustomizev1scaffolds.NewAPIScaffolder(s.config,
		s.resource, true)
	kustomizeV1Scaffolder.InjectFS(s.fs)
	if err := kustomizeV1Scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("error scaffolding kustomize files for the APIs: %v", err)
	}
	return nil
}

func (s *apiScaffolder) scaffoldCreateAPIFromGolang() error {
	// Now we need call the kustomize/v1 plugin to do its scaffolds when we create a new API
	// todo: when we have the go/v4-alpha plugin we will also need to check what is the plugin used
	// in the Project layout to know if we should use kustomize/v1 OR kustomize/v2-alpha

	golangV3Scaffolder := golangv3scaffolds.NewAPIScaffolder(s.config,
		s.resource, true)
	golangV3Scaffolder.InjectFS(s.fs)
	return golangV3Scaffolder.Scaffold()
}

const containerTemplate = `Containers: []corev1.Container{{
						Image: "%s",
						Name:  "%s",
						ImagePullPolicy: corev1.PullIfNotPresent,
						// Ensure restrictive context for the container
						// More info: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &[]bool{true}[0],
							RunAsUser: &[]int64{1000}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
					}}`

const commandTemplate = `
						Command:         []string{%s},`

const portTemplate = `
						Ports: []corev1.ContainerPort{{
							ContainerPort: m.Spec.ContainerPort,
							Name:          "%s",
						}},`
