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
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
)

// Validate checks the input info provided for the update and populates the cliVersion
func (opts *Update) Validate() error {
	if err := opts.validateGitRepo(); err != nil {
		return fmt.Errorf("failed to validate git repository: %w", err)
	}
	if err := opts.validateFromBranch(); err != nil {
		return fmt.Errorf("failed to validate --from-branch: %w", err)
	}
	if err := opts.validateSemanticVersions(); err != nil {
		return fmt.Errorf("failed to validate the versions: %w", err)
	}
	if err := validateReleaseAvailability(opts.FromVersion); err != nil {
		return fmt.Errorf("unable to find release %s: %w", opts.FromVersion, err)
	}
	if err := validateReleaseAvailability(opts.ToVersion); err != nil {
		return fmt.Errorf("unable to find release %s: %w", opts.ToVersion, err)
	}
	return nil
}

// validateGitRepo verifies if the current directory is a valid Git repository and checks for uncommitted changes.
func (opts *Update) validateGitRepo() error {
	log.Info("Checking if is a git repository")
	gitCmd := exec.Command("git", "rev-parse", "--git-dir")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("not in a git repository")
	}

	log.Info("Checking if branch has uncommitted changes")
	gitCmd = exec.Command("git", "status", "--porcelain")
	output, err := gitCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check branch status: %w", err)
	}
	if len(strings.TrimSpace(string(output))) > 0 {
		return fmt.Errorf("working directory has uncommitted changes. " +
			"Please commit or stash them before updating")
	}
	return nil
}

// validateFromBranch the branch passed to the --from-branch flag
func (opts *Update) validateFromBranch() error {
	// Check if the branch exists
	gitCmd := exec.Command("git", "rev-parse", "--verify", opts.FromBranch)
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("%s branch does not exist locally. "+
			"Run 'git branch -a' to see all available branches",
			opts.FromBranch)
	}
	return nil
}

// validateSemanticVersions the version informed by the user via --from-version flag
func (opts *Update) validateSemanticVersions() error {
	if !semver.IsValid(opts.FromVersion) {
		return fmt.Errorf(" version informed (%s) has invalid semantic version. "+
			"Expect: vX.Y.Z (Ex: v4.5.0)", opts.FromVersion)
	}
	if !semver.IsValid(opts.ToVersion) {
		return fmt.Errorf(" version informed (%s) has invalid semantic version. "+
			"Expect: vX.Y.Z (Ex: v4.5.0)", opts.ToVersion)
	}
	return nil
}

// validateReleaseAvailability will verify if the binary to scaffold from-version flag is available
func validateReleaseAvailability(version string) error {
	url := buildReleaseURL(version)
	resp, err := http.Head(url)
	if err != nil {
		return fmt.Errorf("failed to check binary availability: %w", err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Errorf("failed to close connection: %s", err)
		}
	}()

	switch resp.StatusCode {
	case http.StatusOK:
		log.Infof("Binary version %v is available", version)
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("binary version %s not found. Check versions available in releases",
			version)
	default:
		return fmt.Errorf("unexpected response %d when checking binary availability for version %s",
			resp.StatusCode, version)
	}
}
