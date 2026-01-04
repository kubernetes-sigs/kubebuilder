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
		Long: `Delete scaffolded code and manifests for APIs, webhooks, or plugin features.

Deletes generated files and updates PROJECT configuration automatically.
Code inserted at markers in cmd/main.go is automatically removed when possible.

Examples:
  # Delete an API (webhooks must be deleted first)
  kubebuilder delete api --group crew --version v1 --kind Captain

  # Delete specific webhook type
  kubebuilder delete webhook --group crew --version v1 --kind Captain --defaulting

  # Delete all webhook types for a resource
  kubebuilder delete webhook --group crew --version v1 --kind Captain

  # Delete API created with additional plugins
  kubebuilder delete api --group crew --version v1 --kind Captain --plugins=deploy-image/v1-alpha

  # Delete optional plugin features
  kubebuilder delete --plugins=grafana/v1-alpha
  kubebuilder delete --plugins=helm/v2-alpha
`,
		RunE: c.deleteWithPlugins,
	}

	return cmd
}

// deleteWithPlugins handles delete command when --plugins flag is used for optional plugins
func (c CLI) deleteWithPlugins(_ *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("delete requires a subcommand: api or webhook")
	}

	// Filter to ONLY plugins that explicitly support deletion via Edit interface
	// (optional plugins like helm, grafana, autoupdate use Edit with --delete flag internally)
	deletePlugins := []plugin.Plugin{}
	for _, p := range c.resolvedPlugins {
		if deleteSupportPlugin, ok := p.(plugin.HasDeleteSupport); ok && deleteSupportPlugin.SupportsDelete() {
			deletePlugins = append(deletePlugins, p)
		}
	}

	if len(deletePlugins) == 0 {
		return fmt.Errorf("delete requires a subcommand (api, webhook) or " +
			"use with optional plugins (helm, grafana, autoupdate)")
	}

	// Temporarily replace resolvedPlugins with ONLY delete-supporting plugins
	// This ensures non-delete plugins in the chain don't receive the --delete flag
	originalPlugins := c.resolvedPlugins
	c.resolvedPlugins = deletePlugins
	defer func() { c.resolvedPlugins = originalPlugins }()

	// Forward to edit command with --delete flag for optional plugin cleanup
	editCmd := c.newEditCmd()
	editCmd.SetArgs(append(args, "--delete"))
	if err := editCmd.Execute(); err != nil {
		return fmt.Errorf("failed to delete optional plugin features: %w", err)
	}

	return nil
}
