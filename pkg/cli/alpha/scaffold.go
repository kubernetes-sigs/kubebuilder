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

package alpha

import (
	"fmt"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v3/pkg/config/store"
	"sigs.k8s.io/kubebuilder/v3/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
)

// NewScaffoldCommand return a new scaffold command
func NewScaffoldCommand() *cobra.Command {
	var projectConfig, outputPath string
	scaffoldCmd := &cobra.Command{
		Use:   "scaffold",
		Short: "Re-scaffold an existing kuberbuilder project",
		Long: `This command is a helper for you upgrade your project to the latest versions scaffold.
		
		It will:
			- Create a new directory name after the project in the current working directory or at the output path
			- Re-generate the whole project based on the Project file data

		You will still need to move your code and other customizations over.`,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return validate(projectConfig, outputPath)
		},
		Run: func(cmd *cobra.Command, args []string) {
			cwd, err := os.Getwd()
			if err != nil {
				log.Fatal(err)
			}

			projectPath := getProjectPath(cwd, projectConfig)
			store := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
			if err = store.LoadFrom(projectPath); err != nil {
				log.Fatal(err)
			}

			outputDirectory := getDefaultOutputPath(cwd, store.Config().GetProjectName(), outputPath)
			// create output directory
			if err = os.MkdirAll(outputDirectory, os.ModePerm); err != nil {
				log.Fatal(err)
			}
			// use the new directory to set up the new project
			if err = os.Chdir(outputDirectory); err != nil {
				log.Fatal(err)
			}
			// change back to the cwd after completion
			defer func() {
				_ = os.Chdir(cwd)
			}()

			// init project with plugins
			if err = kubebuilderInit(store); err != nil {
				log.Fatal(err)
			}
			// call edit subcommands
			if err = kubebuilderEdit(store); err != nil {
				log.Fatal(err)
			}
			// recreate api/webhooks
			if err = kubebuilderCreate(store); err != nil {
				log.Fatal(err)
			}
		},
	}
	scaffoldCmd.Flags().StringVar(&projectConfig, "project-config", "",
		"path to a kubebuilder project file if not in the current working directory")
	scaffoldCmd.Flags().StringVar(&outputPath, "output", "",
		"path to output the scaffolding. defaults to the current working directory")

	return scaffoldCmd
}

func kubebuilderCreate(store store.Store) error {
	resources, err := store.Config().GetResources()
	if err != nil {
		return err
	}

	var webhooks []resource.Resource
	for i, r := range resources {
		if r.Webhooks != nil {
			// you cannot create a webhook without an API, create them after
			webhooks = append(webhooks, resources[i])
		} else {
			if err = createAPI(r); err != nil {
				return err
			}
		}
	}

	for _, w := range webhooks {
		if err = createWebhook(w); err != nil {
			return err
		}
	}
	return nil
}

func createAPI(resource resource.Resource) error {
	if resource.API == nil || resource.API.IsEmpty() {
		return nil
	}
	var args []string
	args = append(args, "create")
	args = append(args, "api")
	args = append(args, "--resource")
	args = append(args, "--controller")
	if resource.API.Namespaced {
		args = append(args, "--namespaced")
	}

	args = append(args, getAPIGVK(resource)...)
	return util.RunCmd("kubebuilder create api", "kubebuilder", args...)
}

func getAPIGVK(resource resource.Resource) []string {
	var args []string
	if len(resource.Plural) > 0 {
		args = append(args, "--plural")
		args = append(args, resource.Plural)
	}

	if len(resource.Group) > 0 {
		args = append(args, "--group")
		args = append(args, resource.Group)
	}

	if len(resource.Version) > 0 {
		args = append(args, "--version")
		args = append(args, resource.Version)
	}

	if len(resource.Kind) > 0 {
		args = append(args, "--kind")
		args = append(args, resource.Kind)
	}
	return args
}

func createWebhook(resource resource.Resource) error {
	if resource.Webhooks == nil || resource.Webhooks.IsEmpty() {
		return nil
	}
	var args []string
	args = append(args, "create")
	args = append(args, "webhook")

	if resource.HasConversionWebhook() {
		args = append(args, "--conversion")
	}

	if resource.HasValidationWebhook() {
		args = append(args, "--programmatic-validation")
	}

	if resource.HasDefaultingWebhook() {
		args = append(args, "--defaulting")
	}

	args = append(args, getAPIGVK(resource)...)
	return util.RunCmd("kubebuilder create webhook", "kubebuilder", args...)
}

func kubebuilderEdit(store store.Store) error {
	if store.Config().IsMultiGroup() {
		args := []string{"edit", "--multigroup"}
		return util.RunCmd("kubebuilder edit", "kubebuilder", args...)
	}
	return nil
}

func kubebuilderInit(store store.Store) error {
	var args []string
	args = append(args, "init")
	args = append(args, generateInitArgs(store)...)
	return util.RunCmd("kubebuilder init", "kubebuilder", args...)
}

func generateInitArgs(store store.Store) []string {
	var args []string
	// add existing plugins
	plugins := store.Config().GetPluginChain()
	if len(plugins) > 0 {
		args = append(args, "--plugins")
	}
	for i, _ := range plugins {
		args = append(args, plugins[i])
	}

	domain := store.Config().GetDomain()
	if len(domain) > 0 {
		args = append(args, "--domain")
		args = append(args, domain)
	}

	return args
}

func validate(outputPath string, projectConfig string) error {
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

	return checkPathForKubebuilder()
}

func checkPathForKubebuilder() error {
	_, err := exec.LookPath("kubebuilder")
	return err
}

func getProjectPath(cwd string, overrideProjectPath string) string {
	if overrideProjectPath == "" {
		return fmt.Sprintf("%s/%s", cwd, yaml.DefaultPath)
	}
	return overrideProjectPath
}

func getDefaultOutputPath(cwd, projectName, overridePath string) string {
	if overridePath == "" {
		return fmt.Sprintf("%s/%s", cwd, projectName)
	}
	return fmt.Sprintf("%s/%s", overridePath, projectName)
}
