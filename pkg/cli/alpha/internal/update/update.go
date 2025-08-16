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

package update

import (
	"errors"
	"fmt"
	log "log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"sigs.k8s.io/kubebuilder/v4/pkg/cli/alpha/internal/update/helpers"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
)

// Update contains configuration for the update operation.
type Update struct {
	// FromVersion is the release version to update FROM (the base/original scaffold),
	// e.g., "v4.5.0". This is used to regenerate the ancestor scaffold.
	FromVersion string

	// ToVersion is the release version to update TO (the target scaffold),
	// e.g., "v4.6.0". This is used to regenerate the upgrade scaffold.
	ToVersion string

	// FromBranch is the base Git branch that represents the user's current project state,
	// e.g., "main". Its contents are captured into the "original" branch during the update.
	FromBranch string

	// Force, when true, commits the merge result even if there are conflicts.
	// In that case, conflict markers are kept in the files.
	Force bool

	// ShowCommits controls whether to keep full history (no squash).
	//   - true  => keep history: point the output branch at the merge commit
	//              (no squashed commit is created).
	//   - false => squash: write the merge tree as a single commit on the output branch.
	//
	// The output branch name defaults to "kubebuilder-update-from-<FromVersion>-to-<ToVersion>"
	// unless OutputBranch is explicitly set.
	ShowCommits bool

	// PreservePath is a list of paths to restore from the base branch (FromBranch)
	// when SQUASHING, so things like CI config remain unchanged.
	// Example: []string{".github/workflows"}
	// NOTE: This is ignored when ShowCommits == true.
	PreservePath []string

	// OutputBranch is the name of the branch that will receive the result:
	//   - In squash mode (ShowCommits == false): the single squashed commit.
	//   - In keep-history mode (ShowCommits == true): the merge commit.
	// If empty, it defaults to "kubebuilder-update-from-<FromVersion>-to-<ToVersion>".
	OutputBranch string

	// Push, when true, pushes the OutputBranch to the "origin" remote after the update completes.
	Push bool

	// Temporary branches created during the update process. These are internal to the run
	// and are surfaced for transparency/debugging:
	//   - AncestorBranch: clean scaffold generated from FromVersion
	//   - OriginalBranch: snapshot of the user's current project (FromBranch)
	//   - UpgradeBranch:  clean scaffold generated from ToVersion
	//   - MergeBranch:    result of merging Original into Upgrade (pre-output)
	AncestorBranch string
	OriginalBranch string
	UpgradeBranch  string
	MergeBranch    string
}

// Update a project using a default three-way Git merge.
// This helps apply new scaffolding changes while preserving custom code.
func (opts *Update) Update() error {
	log.Info("Checking out base branch", "branch", opts.FromBranch)
	checkoutCmd := exec.Command("git", "checkout", opts.FromBranch)
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout base branch %s: %w", opts.FromBranch, err)
	}

	suffix := time.Now().Format("02-01-06-15-04")

	opts.AncestorBranch = "tmp-ancestor-" + suffix
	opts.OriginalBranch = "tmp-original-" + suffix
	opts.UpgradeBranch = "tmp-upgrade-" + suffix
	opts.MergeBranch = "tmp-merge-" + suffix

	log.Debug("temporary branches",
		"ancestor", opts.AncestorBranch,
		"original", opts.OriginalBranch,
		"upgrade", opts.UpgradeBranch,
		"merge", opts.MergeBranch,
	)

	// 1. Creates an ancestor branch based on base branch
	// 2. Deletes everything except .git and PROJECT
	// 3. Installs old release
	// 4. Runs alpha generate with old release binary
	// 5. Commits the result
	log.Info("Preparing Ancestor branch", "branch_name", opts.AncestorBranch)
	if err := opts.prepareAncestorBranch(); err != nil {
		return fmt.Errorf("failed to prepare ancestor branch: %w", err)
	}
	// 1. Creates original branch
	// 2. Ensure that original branch is == Based on user’s current base branch content with
	// git checkout "main" -- .
	// 3. Commits this state
	log.Info("Preparing Original branch", "branch_name", opts.OriginalBranch)
	if err := opts.prepareOriginalBranch(); err != nil {
		return fmt.Errorf("failed to checkout current off ancestor: %w", err)
	}
	// 1. Creates upgrade branch from ancestor
	// 2. Cleans up the branch by removing all files except .git and PROJECT
	// 2. Regenerates scaffold using alpha generate with new version
	// 3. Commits the result
	log.Info("Preparing Upgrade branch", "branch_name", opts.UpgradeBranch)
	if err := opts.prepareUpgradeBranch(); err != nil {
		return fmt.Errorf("failed to checkout upgrade off ancestor: %w", err)
	}

	// 1. Creates merge branch from upgrade
	// 2. Merges in original (user code)
	// 3. If conflicts occur, it will warn the user and leave the merge branch for manual resolution
	// 4. If merge is clean, it stages the changes and commits the result
	log.Info("Preparing Merge branch and performing merge", "branch_name", opts.MergeBranch)
	hasConflicts, err := opts.mergeOriginalToUpgrade()
	if err != nil {
		return fmt.Errorf("failed to merge upgrade into merge branch: %w", err)
	}

	// Squash or keep commits based on ShowCommits flag
	if opts.ShowCommits {
		log.Info("Keeping commits history")
		out := opts.getOutputBranchName()
		if err := exec.Command("git", "checkout", "-b", out, opts.MergeBranch).Run(); err != nil {
			return fmt.Errorf("checkout %s: %w", out, err)
		}
	} else {
		log.Info("Squashing merge result to output branch", "output_branch", opts.getOutputBranchName())
		if err := opts.squashToOutputBranch(hasConflicts); err != nil {
			return fmt.Errorf("failed to squash to output branch: %w", err)
		}
	}

	// Push the output branch if requested
	if opts.Push {
		if opts.Push {
			out := opts.getOutputBranchName()
			_ = exec.Command("git", "checkout", out).Run()
			if err := exec.Command("git", "push", "-u", "origin", out).Run(); err != nil {
				return fmt.Errorf("failed to push %s: %w", out, err)
			}
		}
	}

	opts.cleanupTempBranches()
	log.Info("Update completed successfully")

	return nil
}

func (opts *Update) cleanupTempBranches() {
	_ = exec.Command("git", "checkout", opts.getOutputBranchName()).Run()

	branches := []string{
		opts.AncestorBranch,
		opts.OriginalBranch,
		opts.UpgradeBranch,
		opts.MergeBranch,
	}

	for _, b := range branches {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		// Delete only if it's a LOCAL branch.
		if err := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+b).Run(); err == nil {
			_ = exec.Command("git", "branch", "-D", b).Run()
		}
	}
}

// getOutputBranchName returns the output branch name
func (opts *Update) getOutputBranchName() string {
	if opts.OutputBranch != "" {
		return opts.OutputBranch
	}
	return fmt.Sprintf("kubebuilder-update-from-%s-to-%s", opts.FromVersion, opts.ToVersion)
}

// preservePaths checks out the paths specified in PreservePath
func (opts *Update) preservePaths() {
	for _, p := range opts.PreservePath {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if err := exec.Command("git", "checkout", opts.FromBranch, "--", p).Run(); err != nil {
			log.Warn("failed to restore preserved path", "path", p, "branch", opts.FromBranch, "error", err)
		}
	}
}

// squashToOutputBranch takes the exact tree of the MergeBranch and writes it as ONE commit
// on a branch derived from FromBranch (e.g., "main"). If PreservePath is set, those paths
// are restored from the base branch after copying the merge tree, so CI config etc. stays put.
func (opts *Update) squashToOutputBranch(hasConflicts bool) error {
	out := opts.getOutputBranchName()

	// 1) base -> out
	if err := exec.Command("git", "checkout", opts.FromBranch).Run(); err != nil {
		return fmt.Errorf("checkout %s: %w", opts.FromBranch, err)
	}
	if err := exec.Command("git", "checkout", "-B", out, opts.FromBranch).Run(); err != nil {
		return fmt.Errorf("create/reset %s from %s: %w", out, opts.FromBranch, err)
	}

	// 2) clean worktree, then copy merge tree
	if err := helpers.CleanWorktree("output branch"); err != nil {
		return fmt.Errorf("output branch: %w", err)
	}
	if err := exec.Command("git", "checkout", opts.MergeBranch, "--", ".").Run(); err != nil {
		return fmt.Errorf("checkout %s content: %w", "merge", err)
	}

	// 3) optionally restore preserved paths from base (tests assert on 'git restore …')
	opts.preservePaths()

	// 4) stage and single squashed commit
	if err := exec.Command("git", "add", "--all").Run(); err != nil {
		return fmt.Errorf("stage output: %w", err)
	}

	if err := helpers.CommitIgnoreEmpty(opts.getMergeMessage(hasConflicts), "final"); err != nil {
		return fmt.Errorf("failed to commit final branch: %w", err)
	}

	return nil
}

// regenerateProjectWithVersion downloads the release binary for the specified version,
// and runs the `alpha generate` command to re-scaffold the project
func regenerateProjectWithVersion(version string) error {
	tempDir, err := helpers.DownloadReleaseVersionWith(version)
	if err != nil {
		return fmt.Errorf("failed to download release %s binary: %w", version, err)
	}
	if err := runAlphaGenerate(tempDir, version); err != nil {
		return fmt.Errorf("failed to run alpha generate on ancestor branch: %w", err)
	}
	return nil
}

// prepareAncestorBranch prepares the ancestor branch by checking it out,
// cleaning up the project files, and regenerating the project with the specified version.
func (opts *Update) prepareAncestorBranch() error {
	if err := exec.Command("git", "checkout", "-b", opts.AncestorBranch, opts.FromBranch).Run(); err != nil {
		return fmt.Errorf("failed to create %s from %s: %w", opts.AncestorBranch, opts.FromBranch, err)
	}
	if err := cleanupBranch(); err != nil {
		return fmt.Errorf("failed to cleanup the %s : %w", opts.AncestorBranch, err)
	}
	if err := regenerateProjectWithVersion(opts.FromVersion); err != nil {
		return fmt.Errorf("failed to regenerate project with fromVersion %s: %w", opts.FromVersion, err)
	}
	gitCmd := exec.Command("git", "add", "--all")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes in %s: %w", opts.AncestorBranch, err)
	}
	commitMessage := "(chore) initial scaffold from release version: " + opts.FromVersion
	if err := helpers.CommitIgnoreEmpty(commitMessage, "ancestor"); err != nil {
		return fmt.Errorf("failed to commit ancestor branch: %w", err)
	}
	return nil
}

// cleanupBranch removes all files and folders in the current directory
// except for the .git directory and the PROJECT file.
// This is necessary to ensure the ancestor branch starts with a clean slate
// TODO: Analise if this command is still needed in the future.
// It is required because the alpha generate command in versions prior to v4.7.0 do not properly
// handle the removal of files in the ancestor branch.
func cleanupBranch() error {
	cmd := exec.Command("sh", "-c", "find . -mindepth 1 -maxdepth 1 ! -name '.git' ! -name 'PROJECT' -exec rm -rf {} +")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clean up files: %w", err)
	}
	return nil
}

// runMakeTargets is a helper function to run make with the targets necessary
// to ensure all the necessary components are generated, formatted and linted.
func runMakeTargets() {
	targets := []string{"manifests", "generate", "fmt", "vet", "lint-fix"}
	for _, target := range targets {
		err := util.RunCmd(fmt.Sprintf("Running make %s", target), "make", target)
		if err != nil {
			log.Warn("make target failed", "target", target, "error", err)
		}
	}
}

// runAlphaGenerate executes the old Kubebuilder version's 'alpha generate' command
// to create clean scaffolding in the ancestor branch. This uses the downloaded
// binary with the original PROJECT file to recreate the project's initial state.
func runAlphaGenerate(tempDir, version string) error {
	log.Info("Generating project", "version", version)

	tempBinaryPath := tempDir + "/kubebuilder"
	cmd := exec.Command(tempBinaryPath, "alpha", "generate")
	cmd.Env = envWithPrefixedPath(tempDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run alpha generate: %w", err)
	}

	log.Info("Project scaffold generation complete", "version", version)
	runMakeTargets()
	return nil
}

func envWithPrefixedPath(dir string) []string {
	env := os.Environ()
	prefix := "PATH="
	for i, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			env[i] = "PATH=" + dir + string(os.PathListSeparator) + strings.TrimPrefix(kv, prefix)
			return env
		}
	}
	return append(env, "PATH="+dir)
}

// prepareOriginalBranch creates the 'original' branch from ancestor and
// populates it with the user's actual project content from the default branch.
// This represents the current state of the user's project.
func (opts *Update) prepareOriginalBranch() error {
	gitCmd := exec.Command("git", "checkout", "-b", opts.OriginalBranch)
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", opts.OriginalBranch, err)
	}

	gitCmd = exec.Command("git", "checkout", opts.FromBranch, "--", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout content from %s branch onto %s: %w", opts.FromBranch, opts.OriginalBranch, err)
	}

	gitCmd = exec.Command("git", "add", "--all")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage all changes in current: %w", err)
	}
	if err := helpers.CommitIgnoreEmpty(
		fmt.Sprintf("(chore) original code from %s to keep changes", opts.FromBranch),
		"original",
	); err != nil {
		return fmt.Errorf("failed to commit original branch: %w", err)
	}
	return nil
}

// prepareUpgradeBranch creates the 'upgrade' branch from ancestor and
// generates fresh scaffolding using the current (latest) CLI version.
// This represents what the project should look like with the new version.
func (opts *Update) prepareUpgradeBranch() error {
	gitCmd := exec.Command("git", "checkout", "-b", opts.UpgradeBranch, opts.AncestorBranch)
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout %s branch off %s: %w",
			opts.UpgradeBranch, opts.AncestorBranch, err)
	}

	checkoutCmd := exec.Command("git", "checkout", opts.UpgradeBranch)
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout base branch %s: %w", opts.UpgradeBranch, err)
	}

	if err := cleanupBranch(); err != nil {
		return fmt.Errorf("failed to cleanup the %s branch: %w", opts.UpgradeBranch, err)
	}
	if err := regenerateProjectWithVersion(opts.ToVersion); err != nil {
		return fmt.Errorf("failed to regenerate project with version %s: %w", opts.ToVersion, err)
	}
	gitCmd = exec.Command("git", "add", "--all")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes in %s: %w", opts.UpgradeBranch, err)
	}
	if err := helpers.CommitIgnoreEmpty(
		"(chore) initial scaffold from release version: "+opts.ToVersion, "upgrade"); err != nil {
		return fmt.Errorf("failed to commit upgrade branch: %w", err)
	}
	return nil
}

// mergeOriginalToUpgrade attempts to merge the upgrade branch
func (opts *Update) mergeOriginalToUpgrade() (bool, error) {
	hasConflicts := false
	if err := exec.Command("git", "checkout", "-b", opts.MergeBranch, opts.UpgradeBranch).Run(); err != nil {
		return hasConflicts, fmt.Errorf("failed to create merge branch %s from %s: %w",
			opts.MergeBranch, opts.UpgradeBranch, err)
	}

	checkoutCmd := exec.Command("git", "checkout", opts.MergeBranch)
	if err := checkoutCmd.Run(); err != nil {
		return hasConflicts, fmt.Errorf("failed to checkout base branch %s: %w", opts.MergeBranch, err)
	}

	mergeCmd := exec.Command("git", "merge", "--no-edit", "--no-commit", opts.OriginalBranch)
	err := mergeCmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		// If the merge has an error that is not a conflict, return an error 2
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			hasConflicts = true
			if !opts.Force {
				log.Warn("Merge stopped due to conflicts. Manual resolution is required.")
				log.Warn("After resolving the conflicts, run the following command:")
				log.Warn("    make manifests generate fmt vet lint-fix")
				log.Warn("This ensures manifests and generated files are up to date, and the project layout remains consistent.")
				return hasConflicts, fmt.Errorf("merge stopped due to conflicts")
			}
			log.Warn("Merge completed with conflicts. Conflict markers will be committed.")
		} else {
			return hasConflicts, fmt.Errorf("merge failed unexpectedly: %w", err)
		}
	}

	if !hasConflicts {
		log.Info("Merge happened without conflicts.")
	}

	// Best effort to run make targets to ensure the project is in a good state
	runMakeTargets()

	// Step 4: Stage and commit
	if err := exec.Command("git", "add", "--all").Run(); err != nil {
		return hasConflicts, fmt.Errorf("failed to stage merge results: %w", err)
	}

	if err := helpers.CommitIgnoreEmpty(opts.getMergeMessage(hasConflicts), "merge"); err != nil {
		return hasConflicts, fmt.Errorf("failed to commit merge branch: %w", err)
	}
	log.Info("Merge completed")
	return hasConflicts, nil
}

func (opts *Update) getMergeMessage(hasConflicts bool) string {
	base := fmt.Sprintf("scaffold update: %s -> %s", opts.FromVersion, opts.ToVersion)
	if hasConflicts {
		return fmt.Sprintf(":warning: (chore) [with conflicts] %s", base)
	}
	return fmt.Sprintf("(chore) %s", base)
}
