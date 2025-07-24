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
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
)

// Update contains configuration for the update operation
type Update struct {
	// FromVersion stores the version to update from, e.g., "v4.5.0".
	FromVersion string
	// ToVersion stores the version to update to, e.g., "v4.6.0".
	ToVersion string
	// FromBranch stores the branch to update from, e.g., "main".
	FromBranch string
	// Force commits the update changes even with merge conflicts
	Force bool

	// UpdateBranches
	AncestorBranch string
	OriginalBranch string
	UpgradeBranch  string
	MergeBranch    string
}

// Update a project using a default three-way Git merge.
// This helps apply new scaffolding changes while preserving custom code.
func (opts *Update) Update() error {
	log.Infof("Checking out base branch: %s", opts.FromBranch)
	checkoutCmd := exec.Command("git", "checkout", opts.FromBranch)
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout base branch %s: %w", opts.FromBranch, err)
	}

	suffix := time.Now().Format("02-01-06-15-04")

	opts.AncestorBranch = "tmp-ancestor-" + suffix
	opts.OriginalBranch = "tmp-original-" + suffix
	opts.UpgradeBranch = "tmp-upgrade-" + suffix
	opts.MergeBranch = "tmp-merge-" + suffix

	log.Infof("Using branch names:")
	log.Infof("  Ancestor: %s", opts.AncestorBranch)
	log.Infof("  Original:  %s", opts.OriginalBranch)
	log.Infof("  Upgrade:  %s", opts.UpgradeBranch)
	log.Infof("  Merge:    %s", opts.MergeBranch)

	// 1. Creates an ancestor branch based on base branch
	// 2. Deletes everything except .git and PROJECT
	// 3. Installs old release
	// 4. Runs alpha generate with old release binary
	// 5. Commits the result
	log.Infof("Preparing Ancestor branch with name %s", opts.AncestorBranch)
	if err := opts.prepareAncestorBranch(); err != nil {
		return fmt.Errorf("failed to prepare ancestor branch: %w", err)
	}
	// 1. Creates original branch
	// 2. Ensure that original branch is == Based on user’s current base branch content with
	// git checkout "main" -- .
	// 3. Commits this state
	log.Infof("Preparing Original branch with name %s", opts.OriginalBranch)
	if err := opts.prepareOriginalBranch(); err != nil {
		return fmt.Errorf("failed to checkout current off ancestor: %w", err)
	}
	// 1. Creates upgrade branch from ancestor
	// 2. Cleans up the branch by removing all files except .git and PROJECT
	// 2. Regenerates scaffold using alpha generate with new version
	// 3. Commits the result
	log.Infof("Preparing Upgrade branch with name %s", opts.UpgradeBranch)
	if err := opts.prepareUpgradeBranch(); err != nil {
		return fmt.Errorf("failed to checkout upgrade off ancestor: %w", err)
	}

	// 1. Creates merge branch from upgrade
	// 2. Merges in original (user code)
	// 3. If conflicts occur, it will warn the user and leave the merge branch for manual resolution
	// 4. If merge is clean, it stages the changes and commits the result
	log.Infof("Preparing Merge branch with name %s and performing merge", opts.MergeBranch)
	if err := opts.mergeOriginalToUpgrade(); err != nil {
		return fmt.Errorf("failed to merge upgrade into merge branch: %w", err)
	}
	return nil
}

// regenerateProjectWithVersion downloads the release binary for the specified version,
// and runs the `alpha generate` command to re-scaffold the project
func regenerateProjectWithVersion(version string) error {
	tempDir, err := binaryWithVersion(version)
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
	gitCmd := exec.Command("git", "checkout", "-b", opts.AncestorBranch, opts.FromBranch)
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to create %s from %s: %w", opts.AncestorBranch, opts.FromBranch, err)
	}
	checkoutCmd := exec.Command("git", "checkout", opts.AncestorBranch)
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout base branch %s: %w", opts.AncestorBranch, err)
	}
	if err := cleanupBranch(); err != nil {
		return fmt.Errorf("failed to cleanup the %s : %w", opts.AncestorBranch, err)
	}
	if err := regenerateProjectWithVersion(opts.FromVersion); err != nil {
		return fmt.Errorf("failed to regenerate project with fromVersion %s: %w", opts.FromVersion, err)
	}
	gitCmd = exec.Command("git", "add", "--all")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes in %s: %w", opts.AncestorBranch, err)
	}
	commitMessage := "Clean scaffold from release version:" + opts.FromVersion
	_ = exec.Command("git", "commit", "-m", commitMessage).Run()
	return nil
}

// binaryWithVersion downloads the specified released version from GitHub releases and saves it
// to a temporary directory with executable permissions.
// Returns the temporary directory path containing the binary.
func binaryWithVersion(version string) (string, error) {
	url := buildReleaseURL(version)

	fs := afero.NewOsFs()
	tempDir, err := afero.TempDir(fs, "", "kubebuilder"+version+"-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	binaryPath := tempDir + "/kubebuilder"
	file, err := os.Create(binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to create the binary file: %w", err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Errorf("failed to close the file: %v", err)
		}
	}()

	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download the binary: %w", err)
	}
	defer func() {
		if err = response.Body.Close(); err != nil {
			log.Errorf("failed to close the connection: %v", err)
		}
	}()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download the binary: HTTP %d", response.StatusCode)
	}

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write the binary content to file: %w", err)
	}

	if err := os.Chmod(binaryPath, 0o755); err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}
	return tempDir, nil
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
			log.Warnf("make %s failed: %v", target, err)
		}
	}
}

// runAlphaGenerate executes the old Kubebuilder version's 'alpha generate' command
// to create clean scaffolding in the ancestor branch. This uses the downloaded
// binary with the original PROJECT file to recreate the project's initial state.
func runAlphaGenerate(tempDir, version string) error {
	log.Infof("Generating project with version %s", version)
	// Temporarily modify PATH to use the downloaded Kubebuilder binary
	tempBinaryPath := tempDir + "/kubebuilder"
	originalPath := os.Getenv("PATH")
	tempEnvPath := tempDir + ":" + originalPath

	if err := os.Setenv("PATH", tempEnvPath); err != nil {
		return fmt.Errorf("failed to set temporary PATH: %w", err)
	}

	defer func() {
		if err := os.Setenv("PATH", originalPath); err != nil {
			log.Errorf("failed to restore original PATH: %v", err)
		}
	}()

	// TODO: we need improve the implementation from utils to allow us
	// to pass the path of the binary and use it to run the alpha generate command.
	cmd := exec.Command(tempBinaryPath, "alpha", "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run alpha generate: %w", err)
	}
	log.Info("Successfully ran alpha generate", version)

	// TODO: Analyse if this command is still needed in the future.
	// It was added because the alpha generate command in versions prior to v4.7.0 does
	// not run those commands automatically which will not allow we properly ensure
	// that all manifests, code generation, formatting, and linting are applied to
	// properly do the 3-way merge.
	runMakeTargets()
	return nil
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

	_ = exec.Command("git", "commit", "-m",
		fmt.Sprintf("Add code from %s into %s",
			opts.FromBranch, opts.OriginalBranch)).Run()
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

	_ = exec.Command("git", "commit", "-m", "Clean scaffolding from version "+opts.ToVersion).Run()
	return nil
}

// mergeOriginalToUpgrade attempts to merge the upgrade branch
func (opts *Update) mergeOriginalToUpgrade() error {
	if err := exec.Command("git", "checkout", "-b", opts.MergeBranch, opts.UpgradeBranch).Run(); err != nil {
		return fmt.Errorf("failed to create merge branch %s from %s: %w", opts.MergeBranch, opts.OriginalBranch, err)
	}

	checkoutCmd := exec.Command("git", "checkout", opts.MergeBranch)
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout base branch %s: %w", opts.MergeBranch, err)
	}

	mergeCmd := exec.Command("git", "merge", "--no-edit", "--no-commit", opts.OriginalBranch)
	err := mergeCmd.Run()

	hasConflicts := false
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
				return fmt.Errorf("merge stopped due to conflicts")
			}
			log.Warn("Merge completed with conflicts. Conflict markers will be committed.")
		} else {
			return fmt.Errorf("merge failed unexpectedly: %w", err)
		}
	}

	if !hasConflicts {
		log.Info("Merge happened without conflicts.")
	}

	// Best effort to run make targets to ensure the project is in a good state
	runMakeTargets()

	// Step 4: Stage and commit
	if err := exec.Command("git", "add", "--all").Run(); err != nil {
		return fmt.Errorf("failed to stage merge results: %w", err)
	}

	message := fmt.Sprintf("Merge from %s to %s.", opts.FromVersion, opts.ToVersion)
	if hasConflicts {
		message += " With conflicts - manual resolution required."
	} else {
		message += " Merge happened without conflicts."
	}

	_ = exec.Command("git", "commit", "-m", message).Run()

	return nil
}
