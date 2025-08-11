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

package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Options defines filters and behavior for change detection.
type Options struct {
	RepoRoot           string
	Includes           []string
	IncludeIsRegex     bool
	SkipIfOnlyDocsYAML bool
	BaseEnvVar         string
	HeadEnvVar         string
	ChangedFilesEnvVar string
}

// changedFiles holds a normalized list of changed file paths.
type changedFiles struct {
	files []string
}

// ShouldRun determines whether the current E2E suite should run, returning a boolean,
// a human-readable reason, and an error if one occurred.
func ShouldRun(opts Options) (bool, string, error) {
	validateAndNormalizeOpts(&opts)
	// Check CI environment first.
	if raw := strings.TrimSpace(os.Getenv(opts.ChangedFilesEnvVar)); raw != "" {
		return decide(parseChangedFiles(raw), opts)
	}

	base := os.Getenv(opts.BaseEnvVar)
	head := os.Getenv(opts.HeadEnvVar)
	if head == "" {
		head = "HEAD"
	}

	cwd, headDiffErr := os.Getwd()
	if headDiffErr != nil {
		log.Fatalf("failed to get current working directory: %v", headDiffErr)
	}
	// restore original directory at the end
	defer func(originalDir string) {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			log.Printf("WARNING: failed to restore working directory to %q: %v", originalDir, chdirErr)
		}
	}(cwd)

	// Confirm RepoRoot exists.
	if info, statErr := os.Stat(opts.RepoRoot); statErr != nil {
		return true, "repo root path invalid or inaccessible", fmt.Errorf("stat repo root: %w", statErr)
	} else if !info.IsDir() {
		return true, "repo root path is not a directory", errors.New("repo root not a directory")
	}

	// Resolve base commit SHA if not set.
	if base == "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if fetchErr := gitFetchOriginMaster(ctx, opts.RepoRoot); fetchErr != nil {
			// log warning, but don't fail; fallback handled below
			logWarning(fmt.Sprintf("git fetch origin/master failed: %v", fetchErr))
		}

		b, resolveBaseErr := gitResolveBaseRef(ctx, opts.RepoRoot, head)
		if resolveBaseErr == nil && b != "" {
			base = b
		} else {
			base = head + "~1" // fallback
		}
	}

	// Diff changed files between base and head.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out, baseDiffErr := gitDiffNames(ctx, opts.RepoRoot, base, head)
	if baseDiffErr != nil {
		// fallback to diff head~1. head
		out, headDiffErr = gitDiffNames(ctx, opts.RepoRoot, head+"~1", head)
		if headDiffErr != nil {
			return true, "diff failed; default to run", fmt.Errorf("git diff failed: %w", headDiffErr)
		}
	}

	return decide(parseChangedFiles(string(out)), opts)
}

func validateAndNormalizeOpts(opts *Options) {
	if opts.RepoRoot == "" {
		opts.RepoRoot = "."
	}
	if opts.BaseEnvVar == "" {
		opts.BaseEnvVar = "PULL_BASE_SHA"
	}
	if opts.HeadEnvVar == "" {
		opts.HeadEnvVar = "PULL_PULL_SHA"
	}
	if opts.ChangedFilesEnvVar == "" {
		opts.ChangedFilesEnvVar = "KUBEBUILDER_CHANGED_FILES"
	}
}

func logWarning(msg string) {
	_, err := fmt.Fprintf(os.Stderr, "WARNING: %s\n", msg)
	if err != nil {
		return
	}
}

// parseChangedFiles splits raw changed file data into normalized paths.
func parseChangedFiles(raw string) changedFiles {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, filepath.ToSlash(line))
		}
	}
	return changedFiles{files: files}
}

// decide determines if the suite should run based on changed files and options.
func decide(ch changedFiles, opts Options) (bool, string, error) {
	if len(ch.files) == 0 {
		return true, "no changes detected; running tests", nil
	}

	if opts.SkipIfOnlyDocsYAML && onlyDocsOrYAML(ch.files) {
		return false, "only documentation or YAML files changed; skipping tests", nil
	}

	if len(opts.Includes) == 0 {
		return true, "no include filters specified; running tests", nil
	}

	if opts.IncludeIsRegex {
		pattern := "^(" + strings.Join(opts.Includes, "|") + ")"
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false, "invalid include regex pattern", fmt.Errorf("compile regex %q: %w", pattern, err)
		}

		for _, file := range ch.files {
			if re.MatchString(file) {
				return true, "matched include regex pattern: " + re.String(), nil
			}
		}
		return false, "no files matched include regex patterns", nil
	}

	for _, file := range ch.files {
		for _, include := range opts.Includes {
			if strings.HasPrefix(file, filepath.ToSlash(include)) {
				return true, "matched include prefix: " + include, nil
			}
		}
	}

	return false, "no files matched include prefixes", nil
}

func onlyDocsOrYAML(files []string) bool {
	pattern := `(?i)(^docs/|\.md$|\.markdown$|^\.github/|` +
		`(OWNERS|OWNERS_ALIASES|SECURITY_CONTACTS|LICENSE)(\.md)?$|\.ya?ml$)`
	re := regexp.MustCompile(pattern)
	for _, file := range files {
		if !re.MatchString(file) {
			return false
		}
	}
	return true
}

// gitFetchOriginMaster runs `git fetch origin master --quiet`.
func gitFetchOriginMaster(ctx context.Context, repoRoot string) error {
	cmd := exec.CommandContext(ctx, "git", "fetch", "origin", "master", "--quiet")
	cmd.Dir = repoRoot
	if originFetchErr := cmd.Run(); originFetchErr != nil {
		return fmt.Errorf("git fetch origin master failed: %w", originFetchErr)
	}
	return nil
}

// gitResolveBaseRef returns the merge-base commit SHA of head and origin/master.
func gitResolveBaseRef(ctx context.Context, repoRoot, head string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "--quiet", "origin/master")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil || len(bytes.TrimSpace(out)) == 0 {
		return "", errors.New("origin/master ref not found")
	}

	mergeBaseCmd := exec.CommandContext(ctx, "git", "merge-base", head, "origin/master")
	mergeBaseCmd.Dir = repoRoot
	mbOut, err := mergeBaseCmd.Output()
	if err != nil {
		return "", fmt.Errorf("git merge-base failed: %w", err)
	}

	return strings.TrimSpace(string(mbOut)), nil
}

// gitDiffNames returns the list of changed files between base and head commits.
func gitDiffNames(ctx context.Context, repoRoot, base, head string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", base, head)
	cmd.Dir = repoRoot
	out, outErr := cmd.Output()
	if outErr != nil {
		return nil, fmt.Errorf("git diff failed: %w", outErr)
	}
	return out, nil
}
