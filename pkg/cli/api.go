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

	"sigs.k8s.io/kubebuilder/v3/pkg/cli/internal/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

func (c cli) newCreateAPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Scaffold a Kubernetes API",
		Long: `Scaffold a Kubernetes API.
`,
		RunE: errCmdFunc(
			fmt.Errorf("api subcommand requires an existing project"),
		),
	}

	// Lookup the plugin for projectVersion and bind it to the command.
	c.bindCreateAPI(cmd)
	return cmd
}

// nolint:dupl
func (c cli) bindCreateAPI(cmd *cobra.Command) {
	if len(c.resolvedPlugins) == 0 {
		cmdErr(cmd, fmt.Errorf(noPluginError))
		return
	}

	var createAPIPlugin plugin.CreateAPI
	for _, p := range c.resolvedPlugins {
		tmpPlugin, isValid := p.(plugin.CreateAPI)
		if isValid {
			if createAPIPlugin != nil {
				err := fmt.Errorf("duplicate API creation plugins (%s, %s), use a more specific plugin key",
					plugin.KeyFor(createAPIPlugin), plugin.KeyFor(p))
				cmdErr(cmd, err)
				return
			}
			createAPIPlugin = tmpPlugin
		}
	}

	if createAPIPlugin == nil {
		cmdErr(cmd, fmt.Errorf("resolved plugins do not provide an API creation plugin: %v", c.pluginKeys))
		return
	}

	cfg, err := config.LoadInitialized()
	if err != nil {
		cmdErr(cmd, err)
		return
	}

	subcommand := createAPIPlugin.GetCreateAPISubcommand()
	subcommand.InjectConfig(cfg.Config)
	meta := subcommand.UpdateMetadata(c.metadata())
	subcommand.BindFlags(cmd.Flags())
	if meta.Description != "" {
		cmd.Long = meta.Description
	}
	if meta.Examples != "" {
		cmd.Example = meta.Examples
	}
	cmd.RunE = runECmdFunc(cfg, subcommand,
		fmt.Sprintf("failed to create API with %q", plugin.KeyFor(createAPIPlugin)))
}
