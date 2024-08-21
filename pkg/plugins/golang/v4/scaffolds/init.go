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
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	kustomizecommonv2 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/hack"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/test/e2e"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/test/utils"
)

const (
	// ControllerRuntimeVersion is the kubernetes-sigs/controller-runtime version to be used in the project
	ControllerRuntimeVersion = "v0.19.0"
	// ControllerToolsVersion is the kubernetes-sigs/controller-tools version to be used in the project
	ControllerToolsVersion = "v0.16.1"
	// EnvtestK8SVersion is the k8s version used to do the scaffold
	EnvtestK8SVersion = "1.31.0"

	imageName = "controller:latest"
)

var _ plugins.Scaffolder = &initScaffolder{}

var kustomizeVersion string

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

// getControllerRuntimeReleaseBranch converts the ControllerRuntime semantic versioning string to a
// release branch string. Example input: "v0.17.0" -> Output: "release-0.17"
func getControllerRuntimeReleaseBranch() string {
	v := strings.TrimPrefix(ControllerRuntimeVersion, "v")
	tmp := strings.Split(v, ".")

	if len(tmp) < 2 {
		fmt.Println("Invalid version format. Expected at least major and minor version numbers.")
		return ""
	}
	releaseBranch := fmt.Sprintf("release-%s.%s", tmp[0], tmp[1])
	return releaseBranch
}

// Scaffold implements cmdutil.Scaffolder
func (s *initScaffolder) Scaffold() error {
	log.Println("Writing scaffold for you to edit...")

	// Initialize the machinery.Scaffold that will write the boilerplate file to disk
	// The boilerplate file needs to be scaffolded as a separate step as it is going to
	// be used by the rest of the files, even those scaffolded in this command call.
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
	)

	if s.license != "none" {
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
	} else {
		s.boilerplatePath = ""
		// Initialize the machinery.Scaffold without boilerplate
		scaffold = machinery.NewScaffold(s.fs,
			machinery.WithConfig(s.config),
		)
	}

	// If the KustomizeV2 was used to do the scaffold then
	// we need to ensure that we use its supported Kustomize Version
	// in order to support it
	kustomizev2 := kustomizecommonv2.Plugin{}
	gov4 := "go.kubebuilder.io/v4"
	pluginKeyForKustomizeV2 := plugin.KeyFor(kustomizev2)

	for _, pluginKey := range s.config.GetPluginChain() {
		if pluginKey == pluginKeyForKustomizeV2 || pluginKey == gov4 {
			kustomizeVersion = kustomizecommonv2.KustomizeVersion
			break
		}
	}

	return scaffold.Execute(
		&templates.Main{
			ControllerRuntimeVersion: ControllerRuntimeVersion,
		},
		&templates.GoMod{
			ControllerRuntimeVersion: ControllerRuntimeVersion,
		},
		&templates.GitIgnore{},
		&templates.Makefile{
			Image:                    imageName,
			BoilerplatePath:          s.boilerplatePath,
			ControllerToolsVersion:   ControllerToolsVersion,
			KustomizeVersion:         kustomizeVersion,
			ControllerRuntimeVersion: ControllerRuntimeVersion,
			EnvtestVersion:           getControllerRuntimeReleaseBranch(),
		},
		&templates.Dockerfile{},
		&templates.DockerIgnore{},
		&templates.Readme{},
		&templates.Golangci{},
		&e2e.Test{},
		&e2e.SuiteTest{},
		&utils.Utils{},
		&templates.DevContainer{},
		&templates.DevContainerPostInstallScript{},
	)
}
