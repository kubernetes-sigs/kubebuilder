/*
Copyright 2020 The Kubernetes Authors.

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

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

const apiErrorMsg = "failed to create API"

func (c CLI) newCreateAPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Create a Kubernetes API",
		Long:  `Create a Kubernetes API by writing a Resource definition and/or a Controller.`,
		RunE: errCmdFunc(
			fmt.Errorf("api subcommand requires an existing project"),
		),
	}

	// Show hint message on how to list flags instead of showing file completion
	cmd.ValidArgsFunction = func(
		_ *cobra.Command,
		args []string,
		toComplete string,
	) ([]cobra.Completion, cobra.ShellCompDirective) {
		completions := []cobra.Completion{}
		if len(args) == 0 && toComplete == "" {
			completions = cobra.AppendActiveHelp(completions, "Type '--' and press TAB to list more flags")
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	// In case no plugin was resolved, instead of failing the construction of the CLI, fail the execution of
	// this subcommand. This allows the use of subcommands that do not require resolved plugins like help.
	if len(c.resolvedPlugins) == 0 {
		cmdErr(cmd, noResolvedPluginError{})
		return cmd
	}

	// Obtain the plugin keys and subcommands from the plugins that implement plugin.CreateAPI.
	subcommands := c.filterSubcommands(
		func(p plugin.Plugin) bool {
			_, isValid := p.(plugin.CreateAPI)
			return isValid
		},
		func(p plugin.Plugin) plugin.Subcommand {
			return p.(plugin.CreateAPI).GetCreateAPISubcommand()
		},
	)

	// Verify that there is at least one remaining plugin.
	if len(subcommands) == 0 {
		cmdErr(cmd, noAvailablePluginError{"API creation"})
		return cmd
	}

	c.applySubcommandHooks(cmd, subcommands, apiErrorMsg, false)

	// Append plugin table after metadata updates
	c.appendPluginTable(cmd, func(p plugin.Plugin) bool {
		_, isValid := p.(plugin.CreateAPI)
		return isValid
	}, "Available plugins that support 'create api'")

	return cmd
}
