/*
Copyright 2025 The Kubernetes Authors.
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
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/cli/alpha/internal/update"
)

// NewUpdateCommand creates and returns a new Cobra command for updating Kubebuilder projects.
// This command helps users upgrade their projects to newer versions of Kubebuilder by performing
// a three-way merge between:
// - The original scaffolding (ancestor)
// - The user's current project state (current)
// - The new version's scaffolding (upgrade)
//
// The update process creates multiple Git branches to facilitate the merge and help users
// resolve any conflicts that may arise during the upgrade process.
func NewUpdateCommand() *cobra.Command {
	opts := update.Update{}
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a Kubebuilder project to a newer version",
		Long: `This command upgrades your Kubebuilder project to the latest scaffold layout using a 3-way merge strategy.

It performs the following steps:
  1. Creates an 'ancestor' branch from the version originally used to scaffold the project
  2. Creates a 'current' branch with your project's current state
  3. Creates an 'upgrade' branch using the new version's scaffolding
  4. Attempts a 3-way merge into a 'merge' branch

The process uses Git branches:
  - ancestor: clean scaffold from the original version
  - current: your existing project state
  - upgrade: scaffold from the target version
  - merge: result of the 3-way merge

If conflicts occur during the merge, the command will stop and leave the merge branch for manual resolution.
Use --force to commit conflicts with markers instead. 

Examples:
  # Update from the version specified in the PROJECT file to the latest release
  kubebuilder alpha update

  # Update from a specific version to the latest release
  kubebuilder alpha update --from-version v4.6.0

  # Update from a specific version to an specific release
  kubebuilder alpha update --from-version v4.5.0 --to-version v4.7.0	

  # Force update even with merge conflicts (commit conflict markers)
  kubebuilder alpha update --force

`,

		PreRunE: func(_ *cobra.Command, _ []string) error {
			err := opts.Prepare()
			if err != nil {
				return fmt.Errorf("failed to prepare update: %w", err)
			}
			return opts.Validate()
		},

		Run: func(_ *cobra.Command, _ []string) {
			if err := opts.Update(); err != nil {
				log.Fatalf("Update failed: %s", err)
			}
		},
	}

	updateCmd.Flags().StringVar(&opts.FromVersion, "from-version", "",
		"binary release version to upgrade from. Should match the version used to init the project and be"+
			"a valid release version, e.g., v4.6.0. If not set, "+
			"it defaults to the version specified in the PROJECT file. ")

	updateCmd.Flags().StringVar(&opts.ToVersion, "to-version", "",
		"binary release version to upgrade to. Should be a valid release version, e.g., v4.7.0. "+
			"If not set, it defaults to the latest release version available in the project repository.")

	updateCmd.Flags().StringVar(&opts.FromBranch, "from-branch", "",
		"Git branch to use as current state of the project for the update.")

	updateCmd.Flags().BoolVar(&opts.Force, "force", false,
		"Force the update even if conflicts occur. Conflicted files will include conflict markers, and a "+
			"commit will be created automatically. Ideal for automation (e.g., cronjobs, CI).")

	return updateCmd
}
