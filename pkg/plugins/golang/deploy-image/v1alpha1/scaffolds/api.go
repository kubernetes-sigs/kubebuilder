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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	kustomizev2scaffolds "sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds/internal/templates/api"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds/internal/templates/config/samples"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/deploy-image/v1alpha1/scaffolds/internal/templates/controllers"
	golangv4scaffolds "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds"
)

var _ plugins.Scaffolder = &apiScaffolder{}

// apiScaffolder contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type apiScaffolder struct {
	config    config.Config
	resource  resource.Resource
	image     string
	command   string
	port      string
	runAsUser string

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewDeployImageScaffolder returns a new Scaffolder for declarative
// nolint: lll
func NewDeployImageScaffolder(config config.Config, res resource.Resource, image,
	command, port, runAsUser string,
) plugins.Scaffolder {
	return &apiScaffolder{
		config:    config,
		resource:  res,
		image:     image,
		command:   command,
		port:      port,
		runAsUser: runAsUser,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *apiScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *apiScaffolder) Scaffold() error {
	log.Println("Writing scaffold for you to edit...")

	if err := s.scaffoldCreateAPI(); err != nil {
		return err
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

	if err := scaffold.Execute(
		&samples.CRDSample{Port: s.port},
	); err != nil {
		return fmt.Errorf("error updating config/samples: %v", err)
	}

	controller := &controllers.Controller{
		ControllerRuntimeVersion: golangv4scaffolds.ControllerRuntimeVersion,
	}

	if err := scaffold.Execute(
		controller,
	); err != nil {
		return fmt.Errorf("error scaffolding controller: %v", err)
	}

	if err := s.updateControllerCode(*controller); err != nil {
		return fmt.Errorf("error updating controller: %v", err)
	}

	defaultMainPath := "cmd/main.go"
	if err := s.updateMainByAddingEventRecorder(defaultMainPath); err != nil {
		return fmt.Errorf("error updating main.go: %v", err)
	}

	if err := scaffold.Execute(
		&controllers.ControllerTest{Port: s.port},
	); err != nil {
		return fmt.Errorf("error creating controller/**_controller_test.go: %v", err)
	}

	if err := s.addEnvVarIntoManager(); err != nil {
		return err
	}

	return nil
}

// addEnvVarIntoManager will update the config/manager/manager.yaml by adding
// a new ENV VAR for to store the image informed which will be used in the
// controller to create the Pod for the Kind
func (s *apiScaffolder) addEnvVarIntoManager() error {
	managerPath := filepath.Join("config", "manager", "manager.yaml")
	err := util.ReplaceInFile(managerPath, `env:`, `env:`)
	if err != nil {
		if err := util.InsertCode(managerPath, `name: manager`, `
        env:`); err != nil {
			return fmt.Errorf("error scaffolding env key in config/manager/manager.yaml")
		}
	}

	if err = util.InsertCode(managerPath, `env:`,
		fmt.Sprintf(envVarTemplate, strings.ToUpper(s.resource.Kind), s.image)); err != nil {
		return fmt.Errorf("error scaffolding env key in config/manager/manager.yaml")
	}

	return nil
}

// scaffoldCreateAPI will reuse the code from the kustomize and base golang
// plugins to do the default scaffolds which an API is created
func (s *apiScaffolder) scaffoldCreateAPI() error {
	if err := s.scaffoldCreateAPIFromGolang(); err != nil {
		return fmt.Errorf("error scaffolding golang files for the new API: %v", err)
	}

	if err := s.scaffoldCreateAPIFromKustomize(); err != nil {
		return fmt.Errorf("error scaffolding kustomize manifests for the new API: %v", err)
	}
	return nil
}

// TODO: replace this implementation by creating its own MainUpdater
// which will have its own controller template which set the recorder so that we can use it
// in the reconciliation to create an event inside for the finalizer
func (s *apiScaffolder) updateMainByAddingEventRecorder(defaultMainPath string) error {
	if err := util.InsertCode(
		defaultMainPath,
		fmt.Sprintf(
			`%sReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),`, s.resource.Kind),
		fmt.Sprintf(recorderTemplate, strings.ToLower(s.resource.Kind)),
	); err != nil {
		return fmt.Errorf("error scaffolding event recorder in %s: %v", defaultMainPath, err)
	}

	return nil
}

// updateControllerCode will update the code generate on the template to add the Container information
func (s *apiScaffolder) updateControllerCode(controller controllers.Controller) error {
	if err := util.ReplaceInFile(
		controller.Path,
		"//TODO: scaffold container",
		fmt.Sprintf(containerTemplate, // value for the image
			strings.ToLower(s.resource.Kind), // value for the name of the container
		),
	); err != nil {
		return fmt.Errorf("error scaffolding container in the controller path (%s): %v",
			controller.Path, err)
	}

	// Scaffold the command if informed
	if len(s.command) > 0 {
		// TODO: improve it to be an spec in the sample and api instead so that
		// users can change the values
		var res string
		for _, value := range strings.Split(s.command, ",") {
			res += fmt.Sprintf(" \"%s\",", strings.TrimSpace(value))
		}
		// remove the latest ,
		res = res[:len(res)-1]
		// remove the first space to not fail in the go fmt ./...
		res = strings.TrimLeft(res, " ")

		if err := util.InsertCode(controller.Path, `SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &[]bool{true}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},`, fmt.Sprintf(commandTemplate, res)); err != nil {
			return fmt.Errorf("error scaffolding command in the  controller path (%s): %v",
				controller.Path, err)
		}
	}

	// Scaffold the port if informed
	if len(s.port) > 0 {
		if err := util.InsertCode(
			controller.Path,
			`SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &[]bool{true}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},`,
			fmt.Sprintf(
				portTemplate,
				strings.ToLower(s.resource.Kind),
				strings.ToLower(s.resource.Kind)),
		); err != nil {
			return fmt.Errorf("error scaffolding container port in the controller path (%s): %v",
				controller.Path,
				err)
		}
	}

	if len(s.runAsUser) > 0 {
		if err := util.InsertCode(
			controller.Path,
			`RunAsNonRoot:             &[]bool{true}[0],`,
			fmt.Sprintf(runAsUserTemplate, s.runAsUser),
		); err != nil {
			return fmt.Errorf("error scaffolding user-id in the controller path (%s): %v",
				controller.Path, err)
		}
	}

	return nil
}

func (s *apiScaffolder) scaffoldCreateAPIFromKustomize() error {
	kustomizeScaffolder := kustomizev2scaffolds.NewAPIScaffolder(
		s.config,
		s.resource,
		true,
	)

	kustomizeScaffolder.InjectFS(s.fs)

	if err := kustomizeScaffolder.Scaffold(); err != nil {
		return fmt.Errorf("error scaffolding kustomize files for the APIs: %v", err)
	}

	return nil
}

func (s *apiScaffolder) scaffoldCreateAPIFromGolang() error {
	golangV4Scaffolder := golangv4scaffolds.NewAPIScaffolder(s.config,
		s.resource, true)
	golangV4Scaffolder.InjectFS(s.fs)
	return golangV4Scaffolder.Scaffold()
}

const containerTemplate = `Containers: []corev1.Container{{
						Image:           image,
						Name:            "%s",
						ImagePullPolicy: corev1.PullIfNotPresent,
						// Ensure restrictive context for the container
						// More info: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &[]bool{true}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
					}}`

const runAsUserTemplate = `
							RunAsUser:                &[]int64{%s}[0],`

const commandTemplate = `
						Command: []string{%s},`

const portTemplate = `
						Ports: []corev1.ContainerPort{{
							ContainerPort: %s.Spec.ContainerPort,
							Name:          "%s",
						}},`

const recorderTemplate = `
		Recorder: mgr.GetEventRecorderFor("%s-controller"),`

const envVarTemplate = `
        - name: %s_IMAGE
          value: %s`
