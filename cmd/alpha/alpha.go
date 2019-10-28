/*
Copyright 2018 The Kubernetes Authors.

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

package alpha

import (
	"github.com/spf13/cobra"
	"sigs.k8s.io/kubebuilder/cmd/alpha/webhookv1"
)

// NewAlphaCommand returns alpha subcommand which will be mounted
// at the root command by the caller.
func NewAlphaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alpha",
		Short: "Expose commands which are in experimental or early stages of development",
		Long:  `Command group for commands which are either experimental or in early stages of development`,
		Example: `
# scaffolds webhook server
kubebuilder alpha webhook <params>
`,
	}

	cmd.AddCommand(
		webhookv1.NewWebhookCmd(),
	)
	return cmd
}
