/*
Copyright 2026 The Kubernetes Authors.

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

//nolint:dupl
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

const deleteErrorMsg = "failed to delete plugin features"

func (c CLI) newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete optional plugin features from the project",
		Long:  `Delete optional plugin features from the project.`,
		RunE: errCmdFunc(
			fmt.Errorf("project must be initialized"),
		),
	}

	// In case no plugin was resolved, instead of failing the construction of the CLI, fail the execution of
	// this subcommand. This allows the use of subcommands that do not require resolved plugins like help.
	if len(c.resolvedPlugins) == 0 {
		cmdErr(cmd, noResolvedPluginError{})
		return cmd
	}

	// Obtain the plugin keys and subcommands from the plugins that implement plugin.Delete.
	subcommands := c.filterSubcommands(
		func(p plugin.Plugin) bool {
			_, isValid := p.(plugin.Delete)
			return isValid
		},
		func(p plugin.Plugin) plugin.Subcommand {
			return p.(plugin.Delete).GetDeleteSubcommand()
		},
	)

	// Verify that at least one plugin provides a delete subcommand.
	if len(subcommands) == 0 {
		cmdErr(cmd, noAvailablePluginError{"delete optional plugin features"})
		return cmd
	}

	c.applySubcommandHooks(cmd, subcommands, deleteErrorMsg, false)

	// Append plugin table after metadata updates.
	c.appendPluginTable(cmd, func(p plugin.Plugin) bool {
		_, isValid := p.(plugin.Delete)
		return isValid
	}, "Available plugins that support 'delete'")

	return cmd
}
