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

	"sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

func (c *cli) newEditCmd() *cobra.Command {
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

func (c *cli) newEditContext() plugin.Context {
	ctx := plugin.Context{
		CommandName: c.commandName,
		Description: `This command will edit the project configuration. You can have single or multi group project.`,
	}

	return ctx
}

// nolint:dupl
func (c *cli) bindEdit(ctx plugin.Context, cmd *cobra.Command) {
	var getter plugin.EditPluginGetter
	for _, p := range c.resolvedPlugins {
		tmpGetter, isGetter := p.(plugin.EditPluginGetter)
		if isGetter {
			if getter != nil {
				err := fmt.Errorf("duplicate edit project plugins for project version %q (%s, %s), "+
					"use a more specific plugin key", c.projectVersion, plugin.KeyFor(getter), plugin.KeyFor(p))
				cmdErr(cmd, err)
				return
			}
			getter = tmpGetter
		}
	}

	cfg, err := config.LoadInitialized()
	if err != nil {
		cmdErr(cmd, err)
		return
	}

	if getter == nil {
		err := fmt.Errorf("layout plugin %q does not support a edit project plugin", cfg.Layout)
		cmdErr(cmd, err)
		return
	}

	editProject := getter.GetEditPlugin()
	editProject.InjectConfig(&cfg.Config)
	editProject.BindFlags(cmd.Flags())
	editProject.UpdateContext(&ctx)
	cmd.Long = ctx.Description
	cmd.Example = ctx.Examples
	cmd.RunE = runECmdFunc(cfg, editProject,
		fmt.Sprintf("failed to edit project with version %q", c.projectVersion))

}
