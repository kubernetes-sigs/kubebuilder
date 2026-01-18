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
)

func (c CLI) newDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a Kubernetes API, webhook, or plugin features",
		Long: `Delete scaffolded code and manifests for APIs, webhooks, or plugin features.

Deletes generated files and updates PROJECT configuration automatically.
Code injected at markers in cmd/main.go requires manual removal (instructions provided).

Examples:
  kubebuilder delete api --group <group> --version <version> --kind <kind>
  kubebuilder delete webhook --group <group> --version <version> --kind <kind>
  kubebuilder delete --plugins=helm/v2-alpha
`,
		RunE: errCmdFunc(
			fmt.Errorf("delete requires a subcommand (api, webhook) or --plugins flag"),
		),
	}

	return cmd
}
