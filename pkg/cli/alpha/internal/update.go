package internal

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// Update contains configuration for the update operation
type Update struct {
	// FromVersion specifies which version of Kubebuilder to use for the update.
	// If empty, the version from the PROJECT file will be used.
	FromVersion string
}

// Update performs a complete project update by creating a three-way merge to help users
// upgrade their Kubebuilder projects. The process creates multiple Git branches:
// - ancestor: Clean state with old Kubebuilder version scaffolding
// - current: User's current project state
// - upgrade: New Kubebuilder version scaffolding
// - merge: Attempts to merge upgrade changes into current state
func (opts *Update) Update() error {
	// Load the PROJECT configuration file to get the current CLI version
	projectConfigFile := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := projectConfigFile.LoadFrom(yaml.DefaultPath); err != nil { // TODO: assess if DefaultPath could be renamed to a more self-descriptive name
		return fmt.Errorf("fail to run command: %w", err)
	}

	// Determine which Kubebuilder version to use for the update
	cliVersion := projectConfigFile.Config().GetCliVersion()

	// Allow override of the version from PROJECT file via command line flag
	if opts.FromVersion != "" {
		// Ensure version has 'v' prefix for consistency with GitHub releases
		if !strings.HasPrefix(opts.FromVersion, "v") {
			opts.FromVersion = "v" + opts.FromVersion
		}
		log.Infof("Overriding cliVersion field %s from PROJECT file with --from-version %s", cliVersion, opts.FromVersion)
		cliVersion = opts.FromVersion
	} else {
		log.Infof("Using CLI version from PROJECT file: %s", cliVersion)
	}

	// Download the specific Kubebuilder binary version for generating clean scaffolding
	tempDir, err := opts.downloadKubebuilderBinary(cliVersion)
	if err != nil {
		return fmt.Errorf("failed to download Kubebuilder %s binary: %w", cliVersion, err)
	}
	log.Infof("Downloaded binary kept at %s for debugging purposes", tempDir)

	// Create ancestor branch with clean state for three-way merge
	if err := opts.checkoutAncestorBranch(); err != nil {
		return fmt.Errorf("failed to checkout the ancestor branch: %w", err)
	}

	// Remove all existing files to create a clean slate for re-scaffolding
	if err := opts.cleanUpAncestorBranch(); err != nil {
		return fmt.Errorf("failed to clean up the ancestor branch: %w", err)
	}

	// Generate clean scaffolding using the old Kubebuilder version
	if err := opts.runAlphaGenerate(tempDir, cliVersion); err != nil {
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
func (opts *Update) downloadKubebuilderBinary(version string) (string, error) {
	cliVersion := version

	// Construct GitHub release URL based on current OS and architecture
	url := fmt.Sprintf("https://github.com/kubernetes-sigs/kubebuilder/releases/download/%s/kubebuilder_%s_%s",
		cliVersion, runtime.GOOS, runtime.GOARCH)

	log.Infof("Downloading the Kubebuilder %s binary from: %s", cliVersion, url)

	// Create temporary directory for storing the downloaded binary
	fs := afero.NewOsFs()
	tempDir, err := afero.TempDir(fs, "", "kubebuilder"+cliVersion+"-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	// Create the binary file in the temporary directory
	binaryPath := tempDir + "/kubebuilder"
	file, err := os.Create(binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to create the binary file: %w", err)
	}
	defer file.Close()

	// Download the binary from GitHub releases
	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download the binary: %w", err)
	}
	defer response.Body.Close()

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
	if err := os.Chmod(binaryPath, 0755); err != nil {
		return "", fmt.Errorf("failed to make binary executable: %w", err)
	}

	log.Infof("Kubebuilder version %s succesfully downloaded to %s", cliVersion, binaryPath)

	return tempDir, nil
}

// checkoutAncestorBranch creates and switches to the 'ancestor' branch.
// This branch will serve as the common ancestor for the three-way merge,
// containing clean scaffolding from the old Kubebuilder version.
func (opts *Update) checkoutAncestorBranch() error {
	gitCmd := exec.Command("git", "checkout", "-b", "ancestor")
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
	// Remove all tracked files from the Git repository
	gitCmd := exec.Command("git", "rm", "-rf", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to remove tracked files in ancestor branch: %w", err)
	}
	log.Info("Successfully removed tracked files from ancestor branch")

	// Remove all untracked files and directories
	gitCmd = exec.Command("git", "clean", "-fd")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to clean untracked files: %w", err)
	}
	log.Info("Successfully cleaned untracked files from ancestor branch")

	// Commit the cleanup to establish the clean state
	gitCmd = exec.Command("git", "commit", "-m", "Clean up the ancestor branch")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit the cleanup in ancestor branch: %w", err)
	}
	log.Info("Successfully committed cleanup on ancestor")

	return nil
}

// runAlphaGenerate executes the old Kubebuilder version's 'alpha generate' command
// to create clean scaffolding in the ancestor branch. This uses the downloaded
// binary with the original PROJECT file to recreate the project's initial state.
func (opts *Update) runAlphaGenerate(tempDir, version string) error {
	tempBinaryPath := tempDir + "/kubebuilder"

	// Temporarily modify PATH to use the downloaded Kubebuilder binary
	originalPath := os.Getenv("PATH")
	tempEnvPath := tempDir + ":" + originalPath

	if err := os.Setenv("PATH", tempEnvPath); err != nil {
		return fmt.Errorf("failed to set temporary PATH: %w", err)
	}
	// Restore original PATH when function completes
	defer func() {
		if err := os.Setenv("PATH", originalPath); err != nil {
			log.Errorf("failed to restore original PATH: %w", err)
		}
	}()

	// Prepare the alpha generate command with proper I/O redirection
	cmd := exec.Command(tempBinaryPath, "alpha", "generate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	// Restore the original PROJECT file from master branch to ensure
	// we're using the correct project configuration for scaffolding
	gitCmd := exec.Command("git", "checkout", "master", "--", "PROJECT")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout PROJECT from master")
	}
	log.Info("Succesfully checked out the PROJECT file from master branch")

	// Execute the alpha generate command to create clean scaffolding
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run alpha generate: %w", err)
	}
	log.Info("Successfully ran alpha generate using Kubebuilder ", version)

	// Stage all generated files
	gitCmd = exec.Command("git", "add", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes in ancestor: %w", err)
	}
	log.Info("Successfully staged all changes in ancestor")

	// Commit the re-scaffolded project to the ancestor branch
	gitCmd = exec.Command("git", "commit", "-m", "Re-scaffold in ancestor")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes in ancestor: %w", err)
	}
	log.Info("Successfully commited changes in ancestor")

	return nil
}

// checkoutCurrentOffAncestor creates the 'current' branch from ancestor and
// populates it with the user's actual project content from the master branch.
// This represents the current state of the user's project.
func (opts *Update) checkoutCurrentOffAncestor() error {
	// Create current branch starting from the clean ancestor state
	gitCmd := exec.Command("git", "checkout", "-b", "current", "ancestor")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout current branch off ancestor: %w", err)
	}
	log.Info("Successfully checked out current branch off ancestor")

	// Overlay the user's actual project content from master branch
	gitCmd = exec.Command("git", "checkout", "master", "--", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout content from master onto current: %w", err)
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
	log.Info("Successfully commited changes in current")

	return nil
}

// checkoutUpgradeOffAncestor creates the 'upgrade' branch from ancestor and
// generates fresh scaffolding using the current (latest) Kubebuilder version.
// This represents what the project should look like with the new version.
func (opts *Update) checkoutUpgradeOffAncestor() error {
	// Create upgrade branch starting from the clean ancestor state
	gitCmd := exec.Command("git", "checkout", "-b", "upgrade", "ancestor")
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
	log.Info("Successfully commited changes in upgrade branch")

	return nil
}

// checkoutMergeOffCurrent creates the 'merge' branch from the current branch.
// This branch will be used to attempt automatic merging of upgrade changes
// with the user's current project state.
func (opts *Update) checkoutMergeOffCurrent() error {
	gitCmd := exec.Command("git", "checkout", "-b", "merge", "current")
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

	return nil
}
