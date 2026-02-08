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

func (c CLI) newCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "create",
		SuggestFor: []string{"new"},
		Short:      "Scaffold a Kubernetes API or webhook",
		Long: fmt.Sprintf(`Scaffold a Kubernetes API or webhook.

Available plugins that support 'create' subcommands:

%s
`, c.getPluginTableFilteredForSubcommand(func(p plugin.Plugin) bool {
			_, hasCreateAPI := p.(plugin.CreateAPI)
			_, hasCreateWebhook := p.(plugin.CreateWebhook)
			return hasCreateAPI || hasCreateWebhook
		})),
	}
}
