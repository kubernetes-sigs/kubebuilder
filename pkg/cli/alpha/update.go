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
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kubebuilder/v4/pkg/cli/alpha/internal/update"
	"sigs.k8s.io/kubebuilder/v4/pkg/logging"
)

func init() {
	// Initialize consistent logging for alpha commands
	opts := logging.HandlerOptions{
		SlogOpts: slog.HandlerOptions{
			Level: slog.LevelInfo,
		},
	}
	handler := logging.NewHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// NewUpdateCommand creates and returns a new Cobra command for updating Kubebuilder projects.
func NewUpdateCommand() *cobra.Command {
	opts := update.Update{}
	var gitCfg []string
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update your project to a newer version (3-way merge; squash by default)",
		Long: `Upgrade your project scaffold using a 3-way merge while preserving your code.

The updater uses four temporary branches during the run:
  • ancestor : clean scaffold from the starting version (--from-version)
  • original : snapshot of your current project (--from-branch)
  • upgrade  : scaffold generated with the target version (--to-version)
  • merge    : result of merging original into upgrade (conflicts possible)

Output branch & history:
  • Default: SQUASH the merge result into ONE commit on:
        kubebuilder-update-from-<from-version>-to-<to-version>
  • --show-commits: keep full history (not compatible with --restore-path).

Conflicts:
  • Default: stop on conflicts and leave the merge branch for manual resolution.
  • --force: commit with conflict markers so automation can proceed.

Other options:
  • --restore-path: restore paths from base when squashing (e.g., CI configs).
  • --output-branch: override the output branch name.
  • --push: push the output branch to 'origin' after the update.
  • --git-config: pass per-invocation Git config as -c key=value (repeatable). When not set,
      defaults to -c merge.renameLimit=999999 to improve rename detection during merges.

Defaults:
  • --from-version / --to-version: resolved from PROJECT and the latest release if unset.
  • --from-branch: defaults to 'main' if not specified.`,
		Example: `
  # Update from the version in PROJECT to the latest, stop on conflicts
  kubebuilder alpha update

  # Update from a specific version to latest
  kubebuilder alpha update --from-version v4.6.0

  # Update from v4.5.0 to v4.7.0 and keep conflict markers (automation-friendly)
  kubebuilder alpha update --from-version v4.5.0 --to-version v4.7.0 --force

  # Keep full commit history instead of squashing
  kubebuilder alpha update --from-version v4.5.0 --to-version v4.7.0 --force --show-commits

  # Squash while preserving CI workflows from base (e.g., main)
  kubebuilder alpha update --force --restore-path .github/workflows

  # Show commits into a custom output branch name
  kubebuilder alpha update --force --show-commits --output-branch my-update-branch

  # Run update and push the output branch to origin (works with or without --show-commits)
  kubebuilder alpha update --from-version v4.6.0 --to-version v4.7.0 --force --push

  # Add extra Git configs (no need to re-specify defaults)
  kubebuilder alpha update --git-config merge.conflictStyle=diff3 --git-config rerere.enabled=true
                                          
  # Disable Git config defaults completely, use only custom configs
  kubebuilder alpha update --git-config disable --git-config rerere.enabled=true`,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if opts.ShowCommits && len(opts.RestorePath) > 0 {
				return fmt.Errorf("the --restore-path flag is not supported with --show-commits")
			}

			// Defaults always on unless "disable" is present anywhere
			defaults := []string{"merge.renameLimit=999999", "diff.renameLimit=999999"}

			hasDisable := false
			filtered := make([]string, 0, len(gitCfg))
			for _, v := range gitCfg {
				if v == "disable" {
					hasDisable = true
					continue
				}
				filtered = append(filtered, v)
			}

			if hasDisable {
				// no defaults; only user-provided configs (excluding "disable")
				opts.GitConfig = filtered
			} else {
				// defaults + user configs (user can override by repeating keys)
				opts.GitConfig = append(defaults, filtered...)
			}

			if err := opts.Prepare(); err != nil {
				return fmt.Errorf("failed to prepare update: %w", err)
			}
			return opts.Validate()
		},
		Run: func(_ *cobra.Command, _ []string) {
			if err := opts.Update(); err != nil {
				slog.Error("Update failed", "error", err)
				os.Exit(1)
			}
		},
	}

	updateCmd.Flags().StringVar(&opts.FromVersion, "from-version", "",
		"binary release version to upgrade from. Should match the version used to init the project and be "+
			"a valid release version, e.g., v4.6.0. If not set, it defaults to the version specified in the PROJECT file.")
	updateCmd.Flags().StringVar(&opts.ToVersion, "to-version", "",
		"binary release version to upgrade to. Should be a valid release version, e.g., v4.7.0. "+
			"If not set, it defaults to the latest release version available in the project repository.")
	updateCmd.Flags().StringVar(&opts.FromBranch, "from-branch", "",
		"Git branch to use as current state of the project for the update.")
	updateCmd.Flags().BoolVar(&opts.Force, "force", false,
		"Force the update even if conflicts occur. Conflicted files will include conflict markers, and a "+
			"commit will be created automatically. Ideal for automation (e.g., cronjobs, CI).")
	updateCmd.Flags().BoolVar(&opts.ShowCommits, "show-commits", false,
		"If set, the update will keep the full history instead of squashing into a single commit.")
	updateCmd.Flags().StringArrayVar(&opts.RestorePath, "restore-path", nil,
		"Paths to preserve from the base branch (repeatable). Not supported with --show-commits.")
	updateCmd.Flags().StringVar(&opts.OutputBranch, "output-branch", "",
		"Override the default output branch name (default: kubebuilder-update-from-<from-version>-to-<to-version>).")
	updateCmd.Flags().BoolVar(&opts.Push, "push", false,
		"Push the output branch to the remote repository after the update.")
	updateCmd.Flags().BoolVar(&opts.OpenGhIssue, "open-gh-issue", false,
		"Create a GitHub issue with a pre-filled checklist and compare link after the update completes (requires `gh`).")
	updateCmd.Flags().StringArrayVar(
		&gitCfg,
		"git-config",
		nil,
		"Per-invocation Git config (repeatable). "+
			"Defaults: -c merge.renameLimit=999999 -c diff.renameLimit=999999. "+
			"Your configs are applied on top. To disable defaults, include `--git-config disable`")
	return updateCmd
}
