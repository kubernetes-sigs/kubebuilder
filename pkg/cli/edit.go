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

	"sigs.k8s.io/kubebuilder/v2/pkg/cli/internal/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugin"
)

func (c cli) newEditCmd() *cobra.Command {
	ctx := c.newEditContext()
	cmd := &cobra.Command{
		Use:     "edit",
		Short:   "This command will edit the project configuration",
		Long:    ctx.Description,
		Example: ctx.Examples,
		RunE: errCmdFunc(
			fmt.Errorf("project must be initialized"),
		),
	}

	// Lookup the plugin for projectVersion and bind it to the command.
	c.bindEdit(ctx, cmd)
	return cmd
}

func (c cli) newEditContext() plugin.Context {
	return plugin.Context{
		CommandName: c.commandName,
		Description: `Edit the project configuration.
`,
	}
}

// nolint:dupl
func (c cli) bindEdit(ctx plugin.Context, cmd *cobra.Command) {
	if len(c.resolvedPlugins) == 0 {
		cmdErr(cmd, fmt.Errorf(noPluginError))
		return
	}

	var editPlugin plugin.Edit
	for _, p := range c.resolvedPlugins {
		tmpPlugin, isValid := p.(plugin.Edit)
		if isValid {
			if editPlugin != nil {
				err := fmt.Errorf(
					"duplicate edit project plugins (%s, %s), use a more specific plugin key",
					plugin.KeyFor(editPlugin), plugin.KeyFor(p))
				cmdErr(cmd, err)
				return
			}
			editPlugin = tmpPlugin
		}
	}

	if editPlugin == nil {
		cmdErr(cmd, fmt.Errorf("resolved plugins do not provide a project edit plugin: %v", c.pluginKeys))
		return
	}

	cfg, err := config.LoadInitialized()
	if err != nil {
		cmdErr(cmd, err)
		return
	}

	subcommand := editPlugin.GetEditSubcommand()
	subcommand.InjectConfig(&cfg.Config)
	subcommand.BindFlags(cmd.Flags())
	subcommand.UpdateContext(&ctx)
	cmd.Long = ctx.Description
	cmd.Example = ctx.Examples
	cmd.RunE = runECmdFunc(cfg, subcommand,
		fmt.Sprintf("failed to edit project with %q", plugin.KeyFor(editPlugin)))
}
