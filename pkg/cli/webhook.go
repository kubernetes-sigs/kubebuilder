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

package cli // nolint:dupl

import (
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v2/pkg/cli/internal/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugin"
)

func (c cli) newCreateWebhookCmd() *cobra.Command {
	ctx := c.newWebhookContext()
	cmd := &cobra.Command{
		Use:     "webhook",
		Short:   "Scaffold a webhook for an API resource",
		Long:    ctx.Description,
		Example: ctx.Examples,
		RunE: errCmdFunc(
			fmt.Errorf("webhook subcommand requires an existing project"),
		),
	}

	// Lookup the plugin for projectVersion and bind it to the command.
	c.bindCreateWebhook(ctx, cmd)
	return cmd
}

func (c cli) newWebhookContext() plugin.Context {
	return plugin.Context{
		CommandName: c.commandName,
		Description: `Scaffold a webhook for an API resource.
`,
	}
}

// nolint:dupl
func (c cli) bindCreateWebhook(ctx plugin.Context, cmd *cobra.Command) {
	if len(c.resolvedPlugins) == 0 {
		cmdErr(cmd, fmt.Errorf(noPluginError))
		return
	}

	var createWebhookPlugin plugin.CreateWebhook
	for _, p := range c.resolvedPlugins {
		tmpPlugin, isValid := p.(plugin.CreateWebhook)
		if isValid {
			if createWebhookPlugin != nil {
				err := fmt.Errorf("duplicate webhook creation plugins (%s, %s), use a more specific plugin key",
					plugin.KeyFor(createWebhookPlugin), plugin.KeyFor(p))
				cmdErr(cmd, err)
				return
			}
			createWebhookPlugin = tmpPlugin
		}
	}

	if createWebhookPlugin == nil {
		cmdErr(cmd, fmt.Errorf("resolved plugins do not provide a webhook creation plugin: %v", c.pluginKeys))
		return
	}

	cfg, err := config.LoadInitialized()
	if err != nil {
		cmdErr(cmd, err)
		return
	}

	subcommand := createWebhookPlugin.GetCreateWebhookSubcommand()
	subcommand.InjectConfig(&cfg.Config)
	subcommand.BindFlags(cmd.Flags())
	subcommand.UpdateContext(&ctx)
	cmd.Long = ctx.Description
	cmd.Example = ctx.Examples
	cmd.RunE = runECmdFunc(cfg, subcommand,
		fmt.Sprintf("failed to create webhook with %q", plugin.KeyFor(createWebhookPlugin)))
}
