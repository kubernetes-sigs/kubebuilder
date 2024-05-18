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

package cli //nolint:dupl

import (
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

const webhookErrorMsg = "failed to create webhook"

func (c CLI) newCreateWebhookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Scaffold a webhook for an API resource",
		Long: `Scaffold a webhook for an API resource.
`,
		RunE: errCmdFunc(
			fmt.Errorf("webhook subcommand requires an existing project"),
		),
	}

	// In case no plugin was resolved, instead of failing the construction of the CLI, fail the execution of
	// this subcommand. This allows the use of subcommands that do not require resolved plugins like help.
	if len(c.resolvedPlugins) == 0 {
		cmdErr(cmd, noResolvedPluginError{})
		return cmd
	}

	// Obtain the plugin keys and subcommands from the plugins that implement plugin.CreateWebhook.
	subcommands := c.filterSubcommands(
		func(p plugin.Plugin) bool {
			_, isValid := p.(plugin.CreateWebhook)
			return isValid
		},
		func(p plugin.Plugin) plugin.Subcommand {
			return p.(plugin.CreateWebhook).GetCreateWebhookSubcommand()
		},
	)

	// Verify that there is at least one remaining plugin.
	if len(subcommands) == 0 {
		cmdErr(cmd, noAvailablePluginError{"webhook creation"})
		return cmd
	}

	c.applySubcommandHooks(cmd, subcommands, webhookErrorMsg, false)

	return cmd
}
