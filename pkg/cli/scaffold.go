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

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewCommand return a new scaffold command
func (c *CLI) newScaffoldCmd() *cobra.Command {
	var projectConfig, outputPath string
	scaffoldCmd := &cobra.Command{
		Use:   "scaffold",
		Short: "Re-scaffold an existing kuberbuilder project",
		// TODO: Better description
		Long: `TODO`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// grab project file
			if err := c.getInfoFromConfigFilePath(projectConfig); err != nil {
				return err
			}
			// add the project name to the output path
			path := fmt.Sprintf("%s/%s", outputPath, c.projectName)
			// create output directory
			if err := os.MkdirAll(path, os.ModePerm); err != nil {
				return err
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			// use the new directory to setup the new project
			if err = os.Chdir(path); err != nil {
				return err
			}
			// change back to the cwd after completion
			defer func() {
				_ = os.Chdir(cwd)
			}()
			// init project with plugins
			if err = c.newInitCmd().Execute(); err != nil {
				return err
			}
			// call edit subcommands
			if err = c.newEditCmd().Execute(); err != nil {
				return err
			}
			// call create apis
			// TODO: check if they were created with specific plugins
			if err = c.newCreateAPICmd().Execute(); err != nil {
				return err
			}
			// call create webhook
			if err = c.newCreateWebhookCmd().Execute(); err != nil {
				return err
			}

			return nil
		},
	}
	scaffoldCmd.Flags().StringVar(&projectConfig, "project-config", "",
		"path to a kubebuilder project file if not in the current working directory")
	scaffoldCmd.Flags().StringVar(&outputPath, "output", "",
		"path to output the scaffolding. defaults to current working directory")

	return scaffoldCmd
}
