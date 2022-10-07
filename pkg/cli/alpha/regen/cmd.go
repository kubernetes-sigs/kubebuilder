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

package regen

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	yamlstore "sigs.k8s.io/kubebuilder/v3/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	"strings"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "regen",
		Short: "Copy the project to the directory old-<project-name> and re-generate the project based on the PROJECT file config",
		Long: `This command is a helper for you upgrade your project to the latest versions scaffold.
		
		It will:
			- Create a new directory named old-<project-name>
			- Then, will remove all content under the project directory
			- Re-generate the whole project based on the Project file data
		Therefore, you can use it to upgrade your project since as a follow up you would need to 
		only compare the project copied to bkp-<project-name> in order to add on top again all
		your code implementation and customizations.
		`,
		PreRunE: validation,
		RunE:    run,
	}

	return cmd
}

func validation() error {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	if _, err := readConfig(currentPath); err != nil {
		return err
	}

	return nil
}

func run() error {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	var config config.Config
	if config, err = readConfig(currentPath); err != nil {
		return err
	}

	output := strings.Replace(currentPath, config.GetProjectName(), "", -1)
	output = filepath.Join(output, fmt.Sprintf("old-%s", config.GetProjectName()))
	if err := os.Mkdir(output, os.ModePerm); err != nil {
		return err
	}

	if err := util.RunCmd(fmt.Sprintf("Copying all files to %s", output), "cp", "-a", currentPath, output); err != nil {
		return err
	}

	if err := util.RunCmd(fmt.Sprintf("Removing all files from current path (%s)", currentPath), "rm", "-rf", "*"); err != nil {
		return err
	}

	// TODO: run command kubebuilder init plugins=<layout values>
	// TODO: run command kubebuilder edit multi-group=true if multi-group is enabled
	// TODO: run kubebuilder create api for all apis scaffold
	// NOTE that if we are scaffolding the API with the plugins delcarative/v1 or deploy-image v1 we also need to add its flags/args
	// TODO: run create webhook if webhooks are created with the flags regards the config found.

	return nil

}

// ReadConfig returns a configuration if a file containing one exists at the
// default path (project root).
func readConfig(path string) (config.Config, error) {
	store := yamlstore.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := store.LoadFrom(path); err != nil {
		return nil, err
	}

	return store.Config(), nil
}
