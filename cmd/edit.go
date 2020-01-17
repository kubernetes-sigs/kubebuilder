/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/project"
	"sigs.k8s.io/yaml"
)

func newEditProjectCmd() *cobra.Command {

	opts := editProjectCmdOptions{}

	editProjectCmd := &cobra.Command{
		Use:   "edit",
		Short: "This command will edit the configuration of the PROJECT file",
		Long:  `This command will edit the configuration of the PROJECT file`,
		Example: `
		# To enable the multigroup layout/support
		kubebuilder edit --multigroup
		
		# To disable the multigroup layout/support
		kubebuilder edit --multigroup=false`,
		Run: func(cmd *cobra.Command, args []string) {
			dieIfNoProject()

			projectInfo, err := scaffold.LoadProjectFile("PROJECT")
			if err != nil {
				log.Fatalf("failed to read the PROJECT file: %v", err)
			}

			if projectInfo.Version != project.Version2 {
				log.Fatalf("kubebuilder multigroup is for project version: 2, the version of this project is: %s \n", projectInfo.Version)
			}

			// Set MultiGroup Option
			projectInfo.MultiGroup = opts.multigroup

			err = saveProjectFile("PROJECT", &projectInfo)
			if err != nil {
				log.Fatalf("error updating project file with resource information : %v", err)
			}
		},
	}

	editProjectCmd.Flags().BoolVar(&opts.multigroup, "multigroup", false,
		"if set as true, then the tool will generate the project files with multigroup layout")

	return editProjectCmd
}

// saveProjectFile saves the given ProjectFile at the given path.
func saveProjectFile(path string, project *input.ProjectFile) error {
	content, err := yaml.Marshal(project)
	if err != nil {
		return fmt.Errorf("error marshalling project info %v", err)
	}
	err = ioutil.WriteFile(path, content, 0666)
	if err != nil {
		return fmt.Errorf("failed to save project file at %s %v", path, err)
	}
	return nil
}

type editProjectCmdOptions struct {
	multigroup bool
}
