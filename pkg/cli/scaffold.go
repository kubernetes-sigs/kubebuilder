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
	"strings"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v3/pkg/config/store/yaml"
)

// NewCommand return a new scaffold command
func (c *CLI) newScaffoldCmd() *cobra.Command {
	var projectConfig, outputPath string
	scaffoldCmd := &cobra.Command{
		Use:   "scaffold",
		Short: "Re-scaffold an existing kuberbuilder project",
		// TODO: Better description
		Long: `TODO`,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if outputPath != "" {
				if _, err := os.Stat(outputPath); os.IsNotExist(err) {
					return fmt.Errorf("output path: %s does not exist. %v", outputPath, err)
				}
			}
			if projectConfig != "" {
				if _, err := os.Stat(projectConfig); os.IsNotExist(err) {
					return fmt.Errorf("project path: %s does not exist. %v", projectConfig, err)
				}
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}

			projectPath := getProjectPath(cwd, projectConfig)
			if err := c.getInfoFromConfigFilePath(projectPath); err != nil {
				return err
			}

			outputDirectory := getDefaultOutputPath(cwd, c.projectName, outputPath)
			// create output directory
			if err := os.MkdirAll(outputDirectory, os.ModePerm); err != nil {
				return err
			}
			// use the new directory to set up the new project
			if err = os.Chdir(outputDirectory); err != nil {
				return err
			}
			// change back to the cwd after completion
			defer func() {
				_ = os.Chdir(cwd)
			}()
			// init project with plugins
			initCmd := c.newInitCmd()
			c.newRootCmd().AddCommand(initCmd)
			initCmd.SetArgs([]string{"--plugins", strings.Join(c.pluginKeys, ",")})
			if err = initCmd.Execute(); err != nil {
				return initCmd.Usage()
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

func getProjectPath(cwd string, overrideProjectPath string) string {
	// By default use the cwd
	if overrideProjectPath == "" {
		return fmt.Sprintf("%s/%s", cwd, yaml.DefaultPath)
	}
	// use the override
	return overrideProjectPath
}

func getDefaultOutputPath(cwd, projectName, overridePath string) string {
	// By default use the cwd
	if overridePath == "" {
		return fmt.Sprintf("%s/%s", cwd, projectName)
	}
	// use the override
	return fmt.Sprintf("%s/%s", overridePath, projectName)
}
