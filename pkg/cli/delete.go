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

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

func (c CLI) newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a Kubernetes API, webhook, or plugin features",
		Long: `Delete a Kubernetes API, webhook, or plugin features.

For resource-specific deletion:
  kubebuilder delete api --group <group> --version <version> --kind <kind>
  kubebuilder delete webhook --group <group> --version <version> --kind <kind>

For project-level plugin deletion:
  kubebuilder delete --plugins=helm/v2-alpha
  kubebuilder delete --plugins=grafana/v1-alpha,autoupdate/v1-alpha
`,
		RunE: errCmdFunc(
			fmt.Errorf("delete requires a subcommand (api, webhook) or --plugins flag"),
		),
	}

	// If plugins are specified, route to Edit subcommand with delete context
	if len(c.resolvedPlugins) > 0 {
		subcommands := c.filterSubcommands(
			func(p plugin.Plugin) bool {
				_, isValid := p.(plugin.Edit)
				return isValid
			},
			func(p plugin.Plugin) plugin.Subcommand {
				return p.(plugin.Edit).GetEditSubcommand()
			},
		)

		if len(subcommands) > 0 {
			c.applySubcommandHooks(cmd, subcommands, "failed to delete plugin features", false)
		}
	}

	return cmd
}
