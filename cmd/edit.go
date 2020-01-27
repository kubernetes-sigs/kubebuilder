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
	"log"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/cmd/internal"
	"sigs.k8s.io/kubebuilder/internal/config"
)

func newEditProjectCmd() *cobra.Command {

	opts := editProjectCmdOptions{}

	editProjectCmd := &cobra.Command{
		Use:   "edit",
		Short: "This command will edit the project configuration",
		Long:  `This command will edit the project configuration`,
		Example: `
		# To enable the multigroup layout/support
		kubebuilder edit --multigroup
		
		# To disable the multigroup layout/support
		kubebuilder edit --multigroup=false`,
		Run: func(cmd *cobra.Command, args []string) {
			internal.DieIfNotConfigured()

			projectConfig, err := config.Load()
			if err != nil {
				log.Fatalf("failed to read the configuration file: %v", err)
			}

			if opts.multigroup {
				if !projectConfig.IsV2() {
					log.Fatalf("kubebuilder multigroup is for project version: 2,"+
						" the version of this project is: %s \n", projectConfig.Version)
				}

				// Set MultiGroup Option
				projectConfig.MultiGroup = true
			}

			err = projectConfig.Save()
			if err != nil {
				log.Fatalf("error updating project file with resource information : %v", err)
			}
		},
	}

	editProjectCmd.Flags().BoolVar(&opts.multigroup, "multigroup", false,
		"if set as true, then the tool will generate the project files with multigroup layout")

	return editProjectCmd
}

type editProjectCmdOptions struct {
	multigroup bool
}
