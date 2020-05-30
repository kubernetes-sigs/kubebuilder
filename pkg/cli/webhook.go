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

	"github.com/blang/semver"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

func (c *cli) newCreateWebhookCmd() *cobra.Command {
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
	ctx := plugin.Context{
		CommandName: c.commandName,
		Description: `Scaffold a webhook for an API resource.
`,
	}
	if !c.configured {
		ctx.Description = fmt.Sprintf("%s\n%s", ctx.Description, runInProjectRootMsg)
	}
	return ctx
}

func (c cli) bindCreateWebhook(ctx plugin.Context, cmd *cobra.Command) {
	var getter plugin.CreateWebhookPluginGetter

	// get the upper plugin version
	tmpVersion, _ := semver.Make("0.0.1")
	for _, p := range c.resolvedPlugins {
		pluginVersion, _ := semver.Make(p.Version())
		if pluginVersion.Compare(tmpVersion) == -1 {
			tmpVersion = pluginVersion
		}
	}

	for _, p := range c.resolvedPlugins {
		tmpGetter, isGetter := p.(plugin.CreateWebhookPluginGetter)
		if isGetter {
			// When has more than one supportable plugin. E.g:
			// - go.kubebuilder.io/v2.0.0 which is supported by V2 and V3
			// - go.kubebuilder.io/v3.0.0 which is supported by V3
			if getter != nil && getter.Version() == tmpVersion.String() {
				// stop when the getter plugin is the upper version found
				break
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
		err := fmt.Errorf("layout plugin %q does not support a webhook creation plugin", cfg.Layout)
		cmdErr(cmd, err)
		return
	}

	createWebhook := getter.GetCreateWebhookPlugin()
	createWebhook.InjectConfig(&cfg.Config)
	createWebhook.BindFlags(cmd.Flags())
	createWebhook.UpdateContext(&ctx)
	cmd.Long = ctx.Description
	cmd.Example = ctx.Examples
	cmd.RunE = runECmdFunc(cfg, createWebhook,
		fmt.Sprintf("failed to create webhook with version %q", c.projectVersion))
}
