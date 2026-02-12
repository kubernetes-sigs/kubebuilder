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
	"errors"
	"fmt"
	log "log/slog"
	"strings"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	kustomizecommonv2 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/cmd"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/github"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/hack"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/test/e2e"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/test/utils"
)

const (
	// GolangciLintVersion is the golangci-lint version to be used in the project
	GolangciLintVersion = "v2.8.0"
	// ControllerRuntimeVersion is the kubernetes-sigs/controller-runtime version to be used in the project
	ControllerRuntimeVersion = "v0.23.1"
	// ControllerToolsVersion is the kubernetes-sigs/controller-tools version to be used in the project
	ControllerToolsVersion = "v0.20.0"

	imageName = "controller:latest"
)

var _ plugins.Scaffolder = &initScaffolder{}

var kustomizeVersion string

type initScaffolder struct {
	config          config.Config
	boilerplatePath string
	license         string
	owner           string
	licenseFile     string
	commandName     string

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewInitScaffolder returns a new Scaffolder for project initialization operations
func NewInitScaffolder(cfg config.Config, license, owner, licenseFile, commandName string) plugins.Scaffolder {
	return &initScaffolder{
		config:          cfg,
		boilerplatePath: hack.DefaultBoilerplatePath,
		license:         license,
		owner:           owner,
		licenseFile:     licenseFile,
		commandName:     commandName,
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
	log.Info("Writing scaffold for you to edit...")

	// Initialize the machinery.Scaffold that will write the boilerplate file to disk
	// The boilerplate file needs to be scaffolded as a separate step as it is going to
	// be used by the rest of the files, even those scaffolded in this command call.
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
	)

	// Create boilerplate if:
	// 1. A custom license file is provided (--license-file overrides --license), OR
	// 2. A built-in license is specified (not "none")
	if s.licenseFile != "" || s.license != "none" {
		bpFile := &hack.Boilerplate{
			License: s.license,
			Owner:   s.owner,
		}

		// If a custom license file is provided, read its content
		if s.licenseFile != "" {
			content, err := afero.ReadFile(afero.NewOsFs(), s.licenseFile)
			if err != nil {
				return fmt.Errorf("failed to read license file %q: %w", s.licenseFile, err)
			}
			bpFile.CustomBoilerplateContent = string(content)
			bpFile.HasCustomBoilerplate = true
			log.Info("Using custom license header file", "file", s.licenseFile)
		}

		bpFile.Path = s.boilerplatePath
		if err := scaffold.Execute(bpFile); err != nil {
			return fmt.Errorf("failed to execute boilerplate: %w", err)
		}

		boilerplate, err := afero.ReadFile(s.fs.FS, s.boilerplatePath)
		if err != nil {
			if errors.Is(err, afero.ErrFileNotFound) {
				log.Warn("unable to find boilerplate file. "+
					"This file is used to generate the license header in the project.\n"+
					"Note that controller-gen will also use this. Ensure that you "+
					"add the license file or configure your project accordingly",
					"file_path", s.boilerplatePath,
					"error", err)
				boilerplate = []byte("")
			} else {
				return fmt.Errorf("failed to load boilerplate: %w", err)
			}
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

	err := scaffold.Execute(
		&cmd.Main{
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
			GolangciLintVersion:      GolangciLintVersion,
			ControllerRuntimeVersion: ControllerRuntimeVersion,
			EnvtestVersion:           getControllerRuntimeReleaseBranch(),
		},
		&templates.Dockerfile{},
		&templates.DockerIgnore{},
		&templates.Readme{CommandName: s.commandName},
		&templates.Agents{CommandName: s.commandName},
		&templates.Golangci{},
		&templates.CustomGcl{
			GolangciLintVersion: GolangciLintVersion,
		},
		&e2e.Test{},
		&e2e.WebhookTestUpdater{WireWebhook: false},
		&e2e.SuiteTest{},
		&github.E2eTestCi{},
		&github.TestCi{},
		&github.LintCi{
			GolangciLintVersion: GolangciLintVersion,
		},
		&utils.Utils{},
		&templates.DevContainer{},
		&templates.DevContainerPostInstallScript{},
	)
	if err != nil {
		return fmt.Errorf("failed to execute init scaffold: %w", err)
	}

	return nil
}
