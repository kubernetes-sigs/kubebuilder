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

package internal

import (
	"fmt"
	"io"
	log "log/slog"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/afero"
	"golang.org/x/mod/semver"

	"sigs.k8s.io/kubebuilder/v4/pkg/config/store"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
)

// Update contains configuration for the update operation
type Update struct {
	// FromVersion specifies which version of Kubebuilder to use for the update.
	// If empty, the version from the PROJECT file will be used.
	FromVersion string
	// FromBranch specifies which branch to use as current when updating
	FromBranch string
	// CliVersion holds the version to be used during the upgrade process
	CliVersion string
	// BinaryURL holds the URL for downloading the specified binary from
	// the releases on GitHub
	BinaryURL string
}

// Update performs a complete project update by creating a three-way merge to help users
// upgrade their Kubebuilder projects. The process creates multiple Git branches:
// - ancestor: Clean state with old Kubebuilder version scaffolding
// - current: User's current project state
// - upgrade: New Kubebuilder version scaffolding
// - merge: Attempts to merge upgrade changes into current state
func (opts *Update) Update() error {
	// Download the specific Kubebuilder binary version for generating clean scaffolding
	tempDir, err := opts.downloadKubebuilderBinary()
	if err != nil {
		return fmt.Errorf("failed to download Kubebuilder %s binary: %w", opts.CliVersion, err)
	}
	log.Info("Downloaded binary kept for debugging purposes", "directory", tempDir)

	// Create ancestor branch with clean state for three-way merge
	if err := opts.checkoutAncestorBranch(); err != nil {
		return fmt.Errorf("failed to checkout the ancestor branch: %w", err)
	}

	// Remove all existing files to create a clean slate for re-scaffolding
	if err := opts.cleanUpAncestorBranch(); err != nil {
		return fmt.Errorf("failed to clean up the ancestor branch: %w", err)
	}

	// Generate clean scaffolding using the old Kubebuilder version
	if err := opts.runAlphaGenerate(tempDir, opts.CliVersion); err != nil {
		return fmt.Errorf("failed to run alpha generate on ancestor branch: %w", err)
	}

	// Create current branch representing user's existing project state
	if err := opts.checkoutCurrentOffAncestor(); err != nil {
		return fmt.Errorf("failed to checkout current off ancestor: %w", err)
	}

	// Create upgrade branch with new Kubebuilder version scaffolding
	if err := opts.checkoutUpgradeOffAncestor(); err != nil {
		return fmt.Errorf("failed to checkout upgrade off ancestor: %w", err)
	}

	// Create merge branch to attempt automatic merging of changes
	if err := opts.checkoutMergeOffCurrent(); err != nil {
		return fmt.Errorf("failed to checkout merge branch off current: %w", err)
	}

	// Attempt to merge upgrade changes into the user's current state
	if err := opts.mergeUpgradeIntoMerge(); err != nil {
		return fmt.Errorf("failed to merge upgrade into merge branch: %w", err)
	}

	return nil
}

// downloadKubebuilderBinary downloads the specified version of Kubebuilder binary
// from GitHub releases and saves it to a temporary directory with executable permissions.
// Returns the temporary directory path containing the binary.
func (opts *Update) downloadKubebuilderBinary() (string, error) {
	// Construct GitHub release URL based on current OS and architecture
	url := opts.BinaryURL

	log.Info("Downloading the Kubebuilder binary", "version", opts.CliVersion, "download_url", url)

	// Create temporary directory for storing the downloaded binary
	fs := afero.NewOsFs()
	tempDir, err := afero.TempDir(fs, "", "kubebuilder"+opts.CliVersion+"-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Create the binary file in the temporary directory
	binaryPath := tempDir + "/kubebuilder"
	file, err := os.Create(binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to create the binary file: %w", err)
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Error("failed to close the file", "error", err)
		}
	}()

	// Download the binary from GitHub releases
	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download the binary: %w", err)
	}
	defer func() {
		if err = response.Body.Close(); err != nil {
			log.Error("failed to close the connection", "error", err)
		}
	}()

	// Check if download was successful
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download the binary: HTTP %d", response.StatusCode)
	}

	// Copy the downloaded content to the local file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write the binary content to file: %w", err)
	}

	// Make the binary executable
	if err := os.Chmod(binaryPath, 0o755); err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	log.Info("Kubebuilder successfully downloaded", "kubebuilder_version", opts.CliVersion, "binary_path", binaryPath)

	return tempDir, nil
}

// checkoutAncestorBranch creates and switches to the 'ancestor' branch.
// This branch will serve as the common ancestor for the three-way merge,
// containing clean scaffolding from the old Kubebuilder version.
func (opts *Update) checkoutAncestorBranch() error {
	gitCmd := exec.Command("git", "checkout", "-b", "tmp-kb-update-ancestor")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to create and checkout ancestor branch: %w", err)
	}
	log.Info("Created and checked out ancestor branch")

	return nil
}

// cleanUpAncestorBranch removes all files from the ancestor branch to create
// a clean state for re-scaffolding. This ensures the ancestor branch only
// contains pure scaffolding without any user modifications.
func (opts *Update) cleanUpAncestorBranch() error {
	log.Info("Cleaning all files and folders except .git and PROJECT")
	// Remove all tracked files from the Git repository
	cmd := exec.Command("find", ".", "-mindepth", "1", "-maxdepth", "1",
		"!", "-name", ".git",
		"!", "-name", "PROJECT",
		"-exec", "rm", "-rf", "{}", "+")
	log.Info("Running cleanup command", "command", cmd.Args)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clean up files in ancestor branch: %w", err)
	}
	log.Info("Successfully cleanup files in ancestor branch")

	// Remove all untracked files and directories
	gitCmd := exec.Command("git", "add", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes in ancestor: %w", err)
	}
	log.Info("Successfully staged changes in ancestor")

	// Commit the cleanup to establish the clean state
	gitCmd = exec.Command("git", "commit", "-m", "Clean up the ancestor branch")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit the cleanup in ancestor branch: %w", err)
	}
	log.Info("Successfully committed cleanup on ancestor")

	return nil
}

// runMakeTargets is a helper function to run make with the targets necessary
// to ensure all the necessary components are generated, formatted and linted.
func runMakeTargets() error {
	targets := []string{"manifests", "generate", "fmt", "vet", "lint-fix"}
	for _, target := range targets {
		log.Info("Running make command", "target", target)
		err := util.RunCmd(fmt.Sprintf("Running make %s", target), "make", target)
		if err != nil {
			return fmt.Errorf("make %s failed: %v", target, err)
		}
	}
	return nil
}

// runAlphaGenerate executes the old Kubebuilder version's 'alpha generate' command
// to create clean scaffolding in the ancestor branch. This uses the downloaded
// binary with the original PROJECT file to recreate the project's initial state.
func (opts *Update) runAlphaGenerate(tempDir, version string) error {
	// Temporarily modify PATH to use the downloaded Kubebuilder binary
	tempBinaryPath := tempDir + "/kubebuilder"
	originalPath := os.Getenv("PATH")
	tempEnvPath := tempDir + ":" + originalPath

	if err := os.Setenv("PATH", tempEnvPath); err != nil {
		return fmt.Errorf("failed to set temporary PATH: %w", err)
	}

	// Restore original PATH when function completes
	defer func() {
		if err := os.Setenv("PATH", originalPath); err != nil {
			log.Error("failed to restore original PATH", "error", err)
		}
	}()

	// Prepare the alpha generate command with proper I/O redirection
	cmd := exec.Command(tempBinaryPath, "alpha", "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	// Execute the alpha generate command to create clean scaffolding
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run alpha generate: %w", err)
	}
	log.Info("Successfully ran alpha generate using Kubebuilder", "version", version)

	// Run make targets to ensure all the necessary components are generated,
	// formatted and linted.
	log.Info("Running 'make manifests generate fmt vet lint-fix'")
	if err := runMakeTargets(); err != nil {
		return fmt.Errorf("failed to run make: %w", err)
	}
	log.Info("Successfully ran make targets in ancestor")

	// Stage all generated files
	gitCmd := exec.Command("git", "add", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes in ancestor: %w", err)
	}
	log.Info("Successfully staged all changes in ancestor")

	// Commit the re-scaffolded project to the ancestor branch
	gitCmd = exec.Command("git", "commit", "-m", "Re-scaffold in ancestor")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes in ancestor: %w", err)
	}
	log.Info("Successfully committed changes in ancestor")

	return nil
}

// checkoutCurrentOffAncestor creates the 'current' branch from ancestor and
// populates it with the user's actual project content from the default branch.
// This represents the current state of the user's project.
func (opts *Update) checkoutCurrentOffAncestor() error {
	// Create current branch starting from the clean ancestor state
	gitCmd := exec.Command("git", "checkout", "-b", "tmp-kb-update-current", "tmp-kb-update-ancestor")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout current branch off ancestor: %w", err)
	}
	log.Info("Successfully checked out current branch off ancestor")

	// Overlay the user's actual project content from default branch
	gitCmd = exec.Command("git", "checkout", opts.FromBranch, "--", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout content from default branch onto current: %w", err)
	}
	log.Info("Successfully checked out content from main onto current branch")

	// Stage all the user's current project content
	gitCmd = exec.Command("git", "add", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage all changes in current: %w", err)
	}
	log.Info("Successfully staged all changes in current")

	// Commit the user's current state to the current branch
	gitCmd = exec.Command("git", "commit", "-m", "Add content from main onto current branch")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	log.Info("Successfully committed changes in current")

	return nil
}

// checkoutUpgradeOffAncestor creates the 'upgrade' branch from ancestor and
// generates fresh scaffolding using the current (latest) Kubebuilder version.
// This represents what the project should look like with the new version.
func (opts *Update) checkoutUpgradeOffAncestor() error {
	// Create upgrade branch starting from the clean ancestor state
	gitCmd := exec.Command("git", "checkout", "-b", "tmp-kb-update-upgrade", "tmp-kb-update-ancestor")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout upgrade branch off ancestor: %w", err)
	}
	log.Info("Successfully checked out upgrade branch off ancestor")

	// Run alpha generate with the current (new) Kubebuilder version
	// This uses the system's installed kubebuilder binary
	cmd := exec.Command("kubebuilder", "alpha", "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run alpha generate on upgrade branch: %w", err)
	}
	log.Info("Successfully ran alpha generate on upgrade branch")

	// Run make targets to ensure all the necessary components are generated,
	// formatted and linted.
	log.Info("Running 'make manifests generate fmt vet lint-fix'")
	if err := runMakeTargets(); err != nil {
		return fmt.Errorf("failed to run make: %w", err)
	}
	log.Info("Successfully ran make targets in upgrade")

	// Stage all the newly generated files
	gitCmd = exec.Command("git", "add", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes on upgrade: %w", err)
	}
	log.Info("Successfully staged all changes in upgrade branch")

	// Commit the new version's scaffolding to the upgrade branch
	gitCmd = exec.Command("git", "commit", "-m", "alpha generate in upgrade branch")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes in upgrade branch: %w", err)
	}
	log.Info("Successfully committed changes in upgrade branch")

	return nil
}

// checkoutMergeOffCurrent creates the 'merge' branch from the current branch.
// This branch will be used to attempt automatic merging of upgrade changes
// with the user's current project state.
func (opts *Update) checkoutMergeOffCurrent() error {
	gitCmd := exec.Command("git", "checkout", "-b", "tmp-kb-update-merge", "tmp-kb-update-current")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout merge branch off current: %w", err)
	}

	return nil
}

// mergeUpgradeIntoMerge attempts to merge the upgrade branch (containing new
// Kubebuilder scaffolding) into the merge branch (containing user's current state).
// If conflicts occur, it warns the user to resolve them manually rather than failing.
func (opts *Update) mergeUpgradeIntoMerge() error {
	gitCmd := exec.Command("git", "merge", "upgrade")
	err := gitCmd.Run()
	if err != nil {
		// Check if the error is due to merge conflicts (exit code 1)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			log.Warn("Merge with conflicts. Please resolve them manually")
			return nil // Don't treat conflicts as fatal errors
		}
		return fmt.Errorf("failed to merge the upgrade branch into the merge branch: %w", err)
	}

	// Run make targets to ensure all the necessary components are generated,
	// formatted and linted.
	log.Info("Running 'make manifests generate fmt vet lint-fix'")
	if err := runMakeTargets(); err != nil {
		return fmt.Errorf("failed to run make: %w", err)
	}
	log.Info("Successfully ran make targets in merge")

	return nil
}

// Validate checks if the user is in a git repository and if the repository is in a clean state.
// It also validates if the version specified by the user is in a valid format and available for
// download as a binary.
func (opts *Update) Validate() error {
	// Validate git repository
	if err := opts.validateGitRepo(); err != nil {
		return fmt.Errorf("failed to validate git repository: %w", err)
	}

	// Validate --from-branch
	if err := opts.validateFromBranch(); err != nil {
		return fmt.Errorf("failed to validate --from-branch: %w", err)
	}

	// Load the PROJECT configuration file
	projectConfigFile, err := opts.loadConfigFile()
	if err != nil {
		return fmt.Errorf("failed to load the PROJECT file: %w", err)
	}

	// Extract the cliVersion field from the PROJECT file
	opts.CliVersion = projectConfigFile.Config().GetCliVersion()

	// Determine which Kubebuilder version to use for the update
	if err := opts.defineFromVersion(); err != nil {
		return fmt.Errorf("failed to define version: %w", err)
	}

	// Validate if the specified version is available as a binary in the releases
	if err := opts.validateBinaryAvailability(); err != nil {
		return fmt.Errorf("failed to validate binary availability: %w", err)
	}

	return nil
}

// Load the PROJECT configuration file to get the current CLI version
func (opts *Update) loadConfigFile() (store.Store, error) {
	projectConfigFile := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	// TODO: assess if DefaultPath could be renamed to a more self-descriptive name
	if err := projectConfigFile.LoadFrom(yaml.DefaultPath); err != nil {
		if _, statErr := os.Stat(yaml.DefaultPath); os.IsNotExist(statErr) {
			return projectConfigFile, fmt.Errorf("no PROJECT file found. Make sure you're in the project root directory")
		}
		return projectConfigFile, fmt.Errorf("fail to load the PROJECT file: %w", err)
	}
	return projectConfigFile, nil
}

// Define the version of the binary to be downloaded
func (opts *Update) defineFromVersion() error {
	// Allow override of the version from PROJECT file via command line flag
	if opts.FromVersion != "" {
		if !semver.IsValid(opts.FromVersion) {
			return fmt.Errorf("invalid semantic version. Expect: vX.Y.Z (Ex: v4.5.0)")
		}
		opts.CliVersion = opts.FromVersion
	}

	if opts.CliVersion == "" {
		return fmt.Errorf("failed to retrieve Kubebuilder version from PROJECT file. Please use --from-version to inform it")
	}

	return nil
}

// Validate if the version specified is available as a binary for download
// from the releases
func (opts *Update) validateBinaryAvailability() error {
	// Ensure version has 'v' prefix for consistency with GitHub releases
	if !strings.HasPrefix(opts.CliVersion, "v") {
		opts.CliVersion = "v" + opts.CliVersion
	}

	// Construct the URL for pulling the binary from GitHub releases
	opts.BinaryURL = fmt.Sprintf("https://github.com/kubernetes-sigs/kubebuilder/releases/download/%s/kubebuilder_%s_%s",
		opts.CliVersion, runtime.GOOS, runtime.GOARCH)

	resp, err := http.Head(opts.BinaryURL)
	if err != nil {
		return fmt.Errorf("failed to check binary availability: %w", err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Error("failed to close connection", "error", err)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		log.Info("Binary version available", "version", opts.CliVersion)
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("binary version %s not found. Check versions available in releases",
			opts.CliVersion)
	default:
		return fmt.Errorf("unexpected response %d when checking binary availability for version %s",
			resp.StatusCode, opts.CliVersion)
	}
}

// Validate if in a git repository with clean state
func (opts *Update) validateGitRepo() error {
	// Check if in a git repository
	gitCmd := exec.Command("git", "rev-parse", "--git-dir")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("not in a git repository")
	}

	// Check if the branch has uncommitted changes
	gitCmd = exec.Command("git", "status", "--porcelain")
	output, err := gitCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check branch status: %w", err)
	}

	if len(strings.TrimSpace(string(output))) > 0 {
		return fmt.Errorf("working directory has uncommitted changes. Please commit or stash them before updating")
	}

	return nil
}

// Validate the branch passed to the --from-branch flag
func (opts *Update) validateFromBranch() error {
	// Set default if not specified
	if opts.FromBranch == "" {
		opts.FromBranch = "main"
	}

	// Check if the branch exists
	gitCmd := exec.Command("git", "rev-parse", "--verify", opts.FromBranch)
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("%s branch does not exist locally. Run 'git branch -a' to see all available branches",
			opts.FromBranch)
	}

	return nil
}
