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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	log "log/slog"
	"os"
	"os/exec"
	"regexp"
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

	// RestorePath is a list of paths to restore from the base branch (FromBranch)
	// when SQUASHING, so things like CI config remain unchanged.
	// Example: []string{".github/workflows"}
	// NOTE: This is ignored when ShowCommits == true.
	RestorePath []string

	// OutputBranch is the name of the branch that will receive the result:
	//   - In squash mode (ShowCommits == false): the single squashed commit.
	//   - In keep-history mode (ShowCommits == true): the merge commit.
	// If empty, it defaults to "kubebuilder-update-from-<FromVersion>-to-<ToVersion>".
	OutputBranch string

	// Push, when true, pushes the OutputBranch to the "origin" remote after the update completes.
	Push bool

	// CommitMessage is the custom merge message to use for successful merges (no conflicts).
	// Set via --merge-message flag.
	// If empty, defaults to: "chore(kubebuilder): update scaffold <from> -> <to>".
	CommitMessage string

	// CommitMessageConflict is the custom conflict message to use when conflicts occur.
	// Set via --conflict-message flag.
	// If empty, defaults to: "chore(kubebuilder): (:warning: manual conflict resolution required)
	// update scaffold <from> -> <to>".
	CommitMessageConflict string

	// OpenGhIssue, when true, automatically creates a GitHub issue after the update
	// completes. The issue includes a pre-filled checklist and a compare link from
	// the base branch (--from-branch) to the output branch. This requires the GitHub
	// CLI (`gh`) to be installed and authenticated in the local environment.
	OpenGhIssue bool

	UseGhModels bool

	// GitConfig holds per-invocation Git settings applied to every `git` command via
	// `git -c key=value`.
	//
	// Examples:
	//   []string{"merge.renameLimit=999999"}         // improve rename detection during merges
	//   []string{"diff.renameLimit=999999"}          // improve rename detection during diffs
	//   []string{"merge.conflictStyle=diff3"}        // show ancestor in conflict markers
	//   []string{"rerere.enabled=true"}              // reuse recorded resolutions
	//
	// Defaults:
	//   When no --git-config flags are provided, the updater adds:
	//     []string{"merge.renameLimit=999999", "diff.renameLimit=999999"}
	//
	// Behavior:
	//   • If one or more --git-config flags are supplied, those values are appended on top of the defaults.
	//   • To disable the defaults entirely, include a literal "disable", for example:
	//       --git-config disable --git-config rerere.enabled=true
	GitConfig []string

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
	// Inform users about GitHub Models if they're opening an issue but not using AI summary
	if opts.OpenGhIssue && !opts.UseGhModels {
		log.Info("Consider enabling GitHub Models to get an AI summary to help with the update")
		log.Info("Use the --use-gh-models flag if your project/organization has permission to use GitHub Models")
	}

	log.Info("Checking out base branch", "branch", opts.FromBranch)
	checkoutCmd := helpers.GitCmd(opts.GitConfig, "checkout", opts.FromBranch)
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
		if err := helpers.GitCmd(opts.GitConfig, "checkout", "-b", out, opts.MergeBranch).Run(); err != nil {
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
			_ = helpers.GitCmd(opts.GitConfig, "checkout", out).Run()
			if err := helpers.GitCmd(opts.GitConfig, "push", "-u", "origin", out).Run(); err != nil {
				return fmt.Errorf("failed to push %s: %w", out, err)
			}
		}
	}

	opts.cleanupTempBranches()
	log.Info("Update completed successfully")

	if opts.OpenGhIssue {
		if err := opts.openGitHubIssue(hasConflicts); err != nil {
			return fmt.Errorf("failed to open GitHub issue: %w", err)
		}
	}

	return nil
}

func (opts *Update) openGitHubIssue(hasConflicts bool) error {
	log.Info("Creating GitHub Issue to track the need to update the project")
	out := opts.getOutputBranchName()

	// Detect repo "owner/name"
	repoCmd := exec.Command("gh", "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	repoBytes, err := repoCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to detect GitHub repository via `gh repo view`: %s", err)
	}
	repo := strings.TrimSpace(string(repoBytes))

	createPRURL := fmt.Sprintf("https://github.com/%s/compare/%s...%s?expand=1", repo, opts.FromBranch, out)
	title := fmt.Sprintf(helpers.IssueTitleTmpl, opts.ToVersion, opts.FromVersion)

	// Skip if an open issue with same title already exists
	checkCmd := exec.Command("gh", "issue", "list",
		"--repo", repo,
		"--state", "open",
		"--search", fmt.Sprintf("in:title \"%s\"", title),
		"--json", "title")
	if checkOut, checkErr := checkCmd.Output(); checkErr == nil && strings.Contains(string(checkOut), title) {
		log.Info("GitHub Issue already exists, skipping creation", "title", title)
		return nil
	}

	// Base issue body
	var body string
	if hasConflicts {
		body = fmt.Sprintf(helpers.IssueBodyTmplWithConflicts, opts.ToVersion, createPRURL, opts.FromVersion, out)
	} else {
		body = fmt.Sprintf(helpers.IssueBodyTmpl, opts.ToVersion, createPRURL, opts.FromVersion, out)
	}

	log.Info("Creating GitHub Issue")
	createCmd := exec.Command("gh", "issue", "create",
		"--repo", repo,
		"--title", title,
		"--body", body,
	)
	createOut, createErr := createCmd.CombinedOutput()
	if createErr != nil {
		return fmt.Errorf("failed to create GitHub issue: %v\n%s", createErr, string(createOut))
	}
	outStr := string(createOut)

	// Try to extract the issue URL from stdout
	issueURL := helpers.FirstURL(outStr)

	// Fallback: query the just-created issue by title
	if issueURL == "" {
		viewCmd := exec.Command("gh", "issue", "list",
			"--repo", repo,
			"--state", "open",
			"--search", fmt.Sprintf("in:title \"%s\"", title),
			"--json", "url",
			"--jq", ".[0].url",
		)
		urlBytes, vErr := viewCmd.Output()
		if vErr != nil {
			log.Warn("could not determine issue URL from gh output", "stdout", outStr, "error", vErr)
		}
		issueURL = strings.TrimSpace(string(urlBytes))
	}
	log.Info("GitHub Issue created to track the update", "url", issueURL, "compare", createPRURL)

	if opts.UseGhModels {
		log.Info("Generating AI summary with gh models")

		if issueURL == "" {
			return fmt.Errorf("issue created but URL could not be determined")
		}

		releaseURL := fmt.Sprintf("https://github.com/kubernetes-sigs/kubebuilder/releases/tag/%s",
			opts.ToVersion)

		ctx := helpers.BuildFullPrompet(
			opts.FromVersion, opts.ToVersion, opts.FromBranch, out,
			createPRURL, releaseURL)

		var outBuf, errBuf bytes.Buffer
		cmd := exec.Command(
			"gh", "models", "run", "openai/gpt-5",
			"--system-prompt", helpers.AiPRPrompt,
		)
		cmd.Stdin = strings.NewReader(ctx)
		cmd.Stdout = &outBuf
		cmd.Stderr = &errBuf
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("gh models run failed: %w\nstderr:\n%s", err, errBuf.String())
		}

		summary := strings.TrimSpace(outBuf.String())
		if summary != "" {
			num := helpers.IssueNumberFromURL(issueURL)
			target := issueURL
			args := make([]string, 4, 7)
			args[0] = "issue"
			args[1] = "comment"
			args[2] = "--repo"
			args[3] = repo
			if num != "" {
				target = num
			}
			args = append(args, target, "--body", summary)
			commentCmd := exec.Command("gh", args...)
			commentCmd.Stdout = os.Stdout
			commentCmd.Stderr = os.Stderr
			if err := commentCmd.Run(); err != nil {
				return fmt.Errorf("failed to add AI summary comment: %s", err)
			}
			log.Info("AI summary comment added to the issue")
		} else {
			log.Warn("AI summary was empty, no comment added")
		}
	}
	return nil
}

func (opts *Update) cleanupTempBranches() {
	_ = helpers.GitCmd(opts.GitConfig, "checkout", opts.getOutputBranchName()).Run()

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
		if err := helpers.GitCmd(opts.GitConfig,
			"show-ref", "--verify", "--quiet", "refs/heads/"+b).Run(); err == nil {
			_ = helpers.GitCmd(opts.GitConfig, "branch", "-D", b).Run()
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

// preservePaths checks out the paths specified in RestorePath
func (opts *Update) preservePaths() {
	for _, p := range opts.RestorePath {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if err := helpers.GitCmd(opts.GitConfig, "checkout", opts.FromBranch, "--", p).Run(); err != nil {
			log.Warn("failed to restore preserved path", "path", p, "branch", opts.FromBranch, "error", err)
		}
	}
}

// squashToOutputBranch takes the exact tree of the MergeBranch and writes it as ONE commit
// on a branch derived from FromBranch (e.g., "main"). If RestorePath is set, those paths
// are restored from the base branch after copying the merge tree, so CI config etc. stays put.
func (opts *Update) squashToOutputBranch(hasConflicts bool) error {
	out := opts.getOutputBranchName()

	// 1) base -> out
	if err := helpers.GitCmd(opts.GitConfig, "checkout", opts.FromBranch).Run(); err != nil {
		return fmt.Errorf("checkout %s: %w", opts.FromBranch, err)
	}
	if err := helpers.GitCmd(opts.GitConfig, "checkout", "-B", out, opts.FromBranch).Run(); err != nil {
		return fmt.Errorf("create/reset %s from %s: %w", out, opts.FromBranch, err)
	}

	// 2) clean worktree, then copy merge tree
	if err := helpers.CleanWorktree("output branch"); err != nil {
		return fmt.Errorf("output branch: %w", err)
	}
	if err := helpers.GitCmd(opts.GitConfig, "checkout", opts.MergeBranch, "--", ".").Run(); err != nil {
		return fmt.Errorf("checkout %s content: %w", "merge", err)
	}

	// 3) optionally restore preserved paths from base (tests assert on 'git restore …')
	opts.preservePaths()

	// 4) stage and single squashed commit
	if err := helpers.GitCmd(opts.GitConfig, "add", "--all").Run(); err != nil {
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
	if err := helpers.GitCmd(opts.GitConfig, "checkout", "-b", opts.AncestorBranch, opts.FromBranch).Run(); err != nil {
		return fmt.Errorf("failed to create %s from %s: %w", opts.AncestorBranch, opts.FromBranch, err)
	}
	if err := cleanupBranch(); err != nil {
		return fmt.Errorf("failed to cleanup the %s : %w", opts.AncestorBranch, err)
	}
	if err := regenerateProjectWithVersion(opts.FromVersion); err != nil {
		return fmt.Errorf("failed to regenerate project with fromVersion %s: %w", opts.FromVersion, err)
	}
	gitCmd := helpers.GitCmd(opts.GitConfig, "add", "--all")
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

// runMakeTargets runs the make targets needed to keep the tree consistent.
// If skipConflicts is true, it avoids running targets that are guaranteed
// to fail noisily when there are unresolved conflicts.
func runMakeTargets(skipConflicts bool) {
	if !skipConflicts {
		for _, t := range []string{"manifests", "generate", "fmt", "vet", "lint-fix"} {
			if err := util.RunCmd(fmt.Sprintf("Running make %s", t), "make", t); err != nil {
				log.Warn("make target failed", "target", t, "error", err)
			}
		}
		return
	}

	// Conflict-aware path: decide what to run based on repo state.
	cs := helpers.DetectConflicts()
	targets := helpers.DecideMakeTargets(cs)

	if cs.Makefile {
		log.Warn("Skipping all make targets because Makefile has merge conflicts")
		return
	}
	if cs.API {
		log.Warn("API conflicts detected; skipping make targets: manifests, generate")
	}
	if cs.AnyGo {
		log.Warn("Go conflicts detected; skipping make targets: fmt, vet, lint-fix")
	}

	if len(targets) == 0 {
		log.Warn("No make targets will be run due to conflicts")
		return
	}

	for _, t := range targets {
		if err := util.RunCmd(fmt.Sprintf("Running make %s", t), "make", t); err != nil {
			log.Warn("make target failed", "target", t, "error", err)
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

	// Capture and reformat subprocess output to match our logging style
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start alpha generate: %w", err)
	}

	// Forward output while reformatting old-style logs
	go forwardAndReformat(stdout, false)
	go forwardAndReformat(stderr, true)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed to run alpha generate: %w", err)
	}

	log.Info("Project scaffold generation complete", "version", version)
	runMakeTargets(false)
	return nil
}

// forwardAndReformat reads from a subprocess stream and reformats old-style logging to new style
func forwardAndReformat(reader io.Reader, isStderr bool) {
	scanner := bufio.NewScanner(reader)

	// Regex to match old-style log format: level=info msg="message"
	logPattern := regexp.MustCompile(`^level=(\w+)\s+msg="?([^"]*)"?(.*)$`)

	for scanner.Scan() {
		line := scanner.Text()

		// Check if this line matches the old log format
		if matches := logPattern.FindStringSubmatch(line); matches != nil {
			level := strings.ToUpper(matches[1])
			message := matches[2]
			rest := matches[3]

			// Convert to new format based on level
			switch level {
			case "INFO":
				log.Info(message + rest)
			case "WARN", "WARNING":
				log.Warn(message + rest)
			case "ERROR":
				log.Error(message + rest)
			case "DEBUG":
				log.Debug(message + rest)
			default:
				// Fallback: print as-is to appropriate stream
				if isStderr {
					fmt.Fprintln(os.Stderr, line)
				} else {
					fmt.Println(line)
				}
			}
		} else {
			// Not a log line, print as-is to appropriate stream
			if isStderr {
				fmt.Fprintln(os.Stderr, line)
			} else {
				fmt.Println(line)
			}
		}
	}
}

func envWithPrefixedPath(dir string) []string {
	env := os.Environ()
	prefix := "PATH="
	for i, kv := range env {
		if after, ok := strings.CutPrefix(kv, prefix); ok {
			env[i] = "PATH=" + dir + string(os.PathListSeparator) + after
			return env
		}
	}
	return append(env, "PATH="+dir)
}

// prepareOriginalBranch creates the 'original' branch from ancestor and
// populates it with the user's actual project content from the default branch.
// This represents the current state of the user's project.
func (opts *Update) prepareOriginalBranch() error {
	gitCmd := helpers.GitCmd(opts.GitConfig, "checkout", "-b", opts.OriginalBranch)
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", opts.OriginalBranch, err)
	}

	gitCmd = helpers.GitCmd(opts.GitConfig, "checkout", opts.FromBranch, "--", ".")
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout content from %s branch onto %s: %w", opts.FromBranch, opts.OriginalBranch, err)
	}

	gitCmd = helpers.GitCmd(opts.GitConfig, "add", "--all")
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
	gitCmd := helpers.GitCmd(opts.GitConfig, "checkout", "-b", opts.UpgradeBranch, opts.AncestorBranch)
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout %s branch off %s: %w",
			opts.UpgradeBranch, opts.AncestorBranch, err)
	}

	checkoutCmd := helpers.GitCmd(opts.GitConfig, "checkout", opts.UpgradeBranch)
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("failed to checkout base branch %s: %w", opts.UpgradeBranch, err)
	}

	if err := cleanupBranch(); err != nil {
		return fmt.Errorf("failed to cleanup the %s branch: %w", opts.UpgradeBranch, err)
	}
	if err := regenerateProjectWithVersion(opts.ToVersion); err != nil {
		return fmt.Errorf("failed to regenerate project with version %s: %w", opts.ToVersion, err)
	}
	gitCmd = helpers.GitCmd(opts.GitConfig, "add", "--all")
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
	if err := helpers.GitCmd(opts.GitConfig, "checkout", "-b", opts.MergeBranch, opts.UpgradeBranch).Run(); err != nil {
		return hasConflicts, fmt.Errorf("failed to create merge branch %s from %s: %w",
			opts.MergeBranch, opts.UpgradeBranch, err)
	}

	checkoutCmd := helpers.GitCmd(opts.GitConfig, "checkout", opts.MergeBranch)
	if err := checkoutCmd.Run(); err != nil {
		return hasConflicts, fmt.Errorf("failed to checkout base branch %s: %w", opts.MergeBranch, err)
	}

	mergeCmd := helpers.GitCmd(opts.GitConfig, "merge", "--no-edit", "--no-commit", opts.OriginalBranch)
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
	runMakeTargets(true)

	// Step 4: Stage and commit
	if err := helpers.GitCmd(opts.GitConfig, "add", "--all").Run(); err != nil {
		return hasConflicts, fmt.Errorf("failed to stage merge results: %w", err)
	}

	if err := helpers.CommitIgnoreEmpty(opts.getMergeMessage(hasConflicts), "merge"); err != nil {
		return hasConflicts, fmt.Errorf("failed to commit merge branch: %w", err)
	}
	log.Info("Merge completed")
	return hasConflicts, nil
}

func (opts *Update) getMergeMessage(hasConflicts bool) string {
	if hasConflicts {
		// Use custom conflict message if provided
		if opts.CommitMessageConflict != "" {
			return opts.CommitMessageConflict
		}
		// Otherwise use default conflict format
		return helpers.ConflictCommitMessage(opts.FromVersion, opts.ToVersion)
	}

	// Use custom commit message if provided
	if opts.CommitMessage != "" {
		return opts.CommitMessage
	}
	return helpers.MergeCommitMessage(opts.FromVersion, opts.ToVersion)
}
