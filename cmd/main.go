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

	"sigs.k8s.io/kubebuilder/v2/cmd/internal"
	"sigs.k8s.io/kubebuilder/v2/cmd/version"
	"sigs.k8s.io/kubebuilder/v2/internal/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/scaffold"
)

// commandOptions represent the types used to implement the different commands
type commandOptions interface {
	// bindFlags binds the command flags to the fields in the options struct
	bindFlags(command *cobra.Command)

	// The following steps define a generic logic to follow when developing new commands. Some steps may be no-ops.
	// - Step 1: load the config failing if expected but not found or if not expected but found
	loadConfig() (*config.Config, error)
	// - Step 2: verify that the command can be run (e.g., go version, project version, arguments, ...)
	validate(*config.Config) error
	// - Step 3: create the Scaffolder instance
	scaffolder(*config.Config) (scaffold.Scaffolder, error)
	// - Step 4: call the Scaffold method of the Scaffolder instance
	// Doesn't need any method
	// - Step 5: finish the command execution
	postScaffold(*config.Config) error
}

// run executes a command
func run(options commandOptions) error {
	// Step 1: load config
	projectConfig, err := options.loadConfig()
	if err != nil {
		return err
	}

	// Step 2: validate
	if err := options.validate(projectConfig); err != nil {
		return err
	}

	// Step 3: create scaffolder
	scaffolder, err := options.scaffolder(projectConfig)
	if err != nil {
		return err
	}

	// Step 4: scaffold
	if err := scaffolder.Scaffold(); err != nil {
		return err
	}

	// Step 5: finish
	if err := options.postScaffold(projectConfig); err != nil {
		return err
	}

	return nil
}

func buildCmdTree() *cobra.Command {
	if internal.ConfiguredAndV1() {
		internal.PrintV1DeprecationWarning()
	}

	// kubebuilder
	rootCmd := newRootCmd()

	// kubebuilder alpha
	alphaCmd := newAlphaCmd()
	// kubebuilder alpha webhook (v1 only)
	if internal.ConfiguredAndV1() {
		alphaCmd.AddCommand(newWebhookCmd())
	}
	// Only add alpha group if it has subcommands
	if alphaCmd.HasSubCommands() {
		rootCmd.AddCommand(alphaCmd)
	}

	// kubebuilder create
	createCmd := newCreateCmd()
	// kubebuilder create api
	createCmd.AddCommand(newAPICmd())
	// kubebuilder create webhook (v2 only)
	if !internal.ConfiguredAndV1() {
		createCmd.AddCommand(newWebhookV2Cmd())
	}
	// Only add create group if it has subcommands
	if createCmd.HasSubCommands() {
		rootCmd.AddCommand(createCmd)
	}

	// kubebuilder edit
	rootCmd.AddCommand(newEditCmd())

	// kubebuilder init
	rootCmd.AddCommand(newInitCmd())

	// kubebuilder update (v1 only)
	if internal.ConfiguredAndV1() {
		rootCmd.AddCommand(newUpdateCmd())
	}

	// kubebuilder version
	rootCmd.AddCommand(version.NewVersionCmd())

	return rootCmd
}

func main() {
	if err := buildCmdTree().Execute(); err != nil {
		log.Fatal(err)
	}
}
