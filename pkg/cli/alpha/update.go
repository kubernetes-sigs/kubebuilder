package alpha

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kubebuilder/v4/pkg/cli/alpha/internal"
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
	opts := internal.Update{}
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a Kubebuilder project to a newer version",
		Long: `Update a Kubebuilder project to a newer version using a three-way merge strategy.

This command helps you upgrade your Kubebuilder project by:
1. Creating a clean ancestor branch with the old version's scaffolding
2. Creating a current branch with your project's current state
3. Creating an upgrade branch with the new version's scaffolding
4. Attempting to merge the changes automatically

The process creates several Git branches to help you manage the upgrade:
- ancestor: Clean scaffolding from the original version
- current: Your project's current state
- upgrade: Clean scaffolding from the new version
- merge: Attempted automatic merge of upgrade into current

If conflicts occur during the merge, you'll need to resolve them manually.

Examples:
  # Update using the version specified in PROJECT file
  kubebuilder alpha update

  # Update from a specific version
  kubebuilder alpha update --from-version v3.0.0

Requirements:
- Must be run from the root of a Kubebuilder project
- Git repository must be clean (no uncommitted changes)
- PROJECT file must exist and contain a valid layout version`,

		// TODO: Add validation to ensure we're in a Kubebuilder project and Git repo is clean
		//	PreRunE: func(_ *cobra.Command, _ []string) error {
		//		return opts.Validate()
		//	},

		Run: func(_ *cobra.Command, _ []string) {
			if err := opts.Update(); err != nil {
				log.Fatalf("Update failed: %s", err)
			}
		},
	}

	// Flag to override the version specified in the PROJECT file
	updateCmd.Flags().StringVar(&opts.FromVersion, "from-version", "",
		"Override the CLI version from PROJECT file. Specify the Kubebuilder version to upgrade from (e.g., 'v3.0.0' or '3.0.0')")

	return updateCmd
}
