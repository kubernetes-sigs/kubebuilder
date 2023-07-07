/*
Copyright 2023 The Kubernetes Authors.
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

package rescaffold

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v3/pkg/config/store"
	"sigs.k8s.io/kubebuilder/v3/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
)

type MigrateOptions struct {
	InputDir  string
	OutputDir string
}

const DefaultOutputDir = "output-dir"

func (opts *MigrateOptions) Rescaffold() error {
	config := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := config.LoadFrom(opts.InputDir); err != nil {
		log.Fatalf("Failed to load PROJECT file %v", err)
	}
	// create output directory
	// nolint: gosec
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory %v", err)
	}
	// use the new directory to set up the new project
	if err := os.Chdir(opts.OutputDir); err != nil {
		log.Fatalf("Failed to change the current working directory %v", err)
	}
	// init project with plugins
	if err := kubebuilderInit(config); err != nil {
		log.Fatalf("Failed to run init subcommand %v", err)
	}
	// call edit subcommands to enable or disable multigroup layout
	if err := kubebuilderEdit(config); err != nil {
		log.Fatalf("Failed to run edit subcommand %v", err)
	}
	return nil
}

func (opts *MigrateOptions) Validate() error {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	// get PROJECT path from command args
	inputPath, err := getInputPath(cwd, opts.InputDir)
	if err != nil {
		log.Fatal(err)
	}
	opts.InputDir = inputPath
	// get output path from command args
	opts.OutputDir, err = getOutputPath(cwd, opts.OutputDir)
	if err != nil {
		log.Fatal(err)
	}
	// check whether the kubebuilder binary is accessible
	_, err = exec.LookPath("kubebuilder")
	return err
}

func getInputPath(currentWorkingDirectory string, inputPath string) (string, error) {
	if inputPath == "" {
		inputPath = currentWorkingDirectory
	}
	projectPath := fmt.Sprintf("%s/%s", inputPath, yaml.DefaultPath)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return "", fmt.Errorf("PROJECT path: %s does not exist. %v", projectPath, err)
	}
	return projectPath, nil
}

func getOutputPath(currentWorkingDirectory, outputPath string) (string, error) {
	if outputPath == "" {
		outputPath = fmt.Sprintf("%s/%s", currentWorkingDirectory, DefaultOutputDir)
	}
	_, err := os.Stat(outputPath)
	if err == nil {
		return "", fmt.Errorf("Output path: %s already exists. %v", outputPath, err)
	}
	if os.IsNotExist(err) {
		return outputPath, nil
	}
	return "", err
}

func kubebuilderInit(store store.Store) error {
	var args []string
	args = append(args, "init")
	args = append(args, getInitArgs(store)...)
	return util.RunCmd("kubebuilder init", "kubebuilder", args...)
}

func kubebuilderEdit(store store.Store) error {
	if store.Config().IsMultiGroup() {
		args := []string{"edit", "--multigroup"}
		return util.RunCmd("kubebuilder edit", "kubebuilder", args...)
	}
	return nil
}

func getInitArgs(store store.Store) []string {
	var args []string
	plugins := store.Config().GetPluginChain()
	if len(plugins) > 0 {
		args = append(args, "--plugins")
		args = append(args, plugins...)
	}
	domain := store.Config().GetDomain()
	if domain != "" {
		args = append(args, "--domain", domain)
	}
	return args
}
