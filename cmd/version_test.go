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

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPseudoVersionFromGitDirExactTag(t *testing.T) {
	repo := initGitRepo(t)

	if _, err := runGitCommand(repo, "tag", "v1.2.3"); err != nil {
		t.Fatalf("tagging repo: %v", err)
	}

	version := pseudoVersionFromGitDir("example.com/module/v1", repo)
	if version != "v1.2.3" {
		t.Fatalf("expected tag version, got %q", version)
	}
}

func TestPseudoVersionFromGitDirAfterTag(t *testing.T) {
	repo := initGitRepo(t)

	if _, err := runGitCommand(repo, "tag", "v1.2.3"); err != nil {
		t.Fatalf("tagging repo: %v", err)
	}
	createCommit(t, repo, "second file", "second change")

	version := pseudoVersionFromGitDir("example.com/module/v1", repo)
	if version == "" {
		t.Fatalf("expected pseudo version, got empty string")
	}

	hash, err := runGitCommand(repo, "rev-parse", "--short=12", "HEAD")
	if err != nil {
		t.Fatalf("retrieving hash: %v", err)
	}
	timestampStr, err := runGitCommand(repo, "show", "-s", "--format=%ct", "HEAD")
	if err != nil {
		t.Fatalf("retrieving timestamp: %v", err)
	}
	seconds, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		t.Fatalf("parsing timestamp: %v", err)
	}
	expected := fmt.Sprintf("v1.2.4-0.%s-%s", time.Unix(seconds, 0).UTC().Format(pseudoVersionTimestampLayout), hash)
	if version != expected {
		t.Fatalf("expected %q, got %q", expected, version)
	}
}

func TestPseudoVersionFromGitDirDirty(t *testing.T) {
	repo := initGitRepo(t)

	if _, err := runGitCommand(repo, "tag", "v1.2.3"); err != nil {
		t.Fatalf("tagging repo: %v", err)
	}
	createCommit(t, repo, "second file", "second change")

	targetFile := filepath.Join(repo, "tracked.txt")
	if err := os.WriteFile(targetFile, []byte("dirty change\n"), 0o644); err != nil {
		t.Fatalf("creating dirty file: %v", err)
	}

	version := pseudoVersionFromGitDir("example.com/module/v1", repo)
	if version != develVersion {
		t.Fatalf("expected %q for dirty repo, got %q", develVersion, version)
	}
}

func TestPseudoVersionFromGitDirWithoutTag(t *testing.T) {
	repo := initGitRepo(t)
	version := pseudoVersionFromGitDir("example.com/module/v4", repo)
	if !strings.HasPrefix(version, "v4.0.0-") {
		t.Fatalf("expected prefix v4.0.0-, got %q", version)
	}
}

func TestGetKubebuilderVersionDirtyString(t *testing.T) {
	t.Cleanup(func() { kubeBuilderVersion = unknown })
	kubeBuilderVersion = "v1.2.3+dirty"
	if got := getKubebuilderVersion(); got != develVersion {
		t.Fatalf("expected %q, got %q", develVersion, got)
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	commands := [][]string{
		{"init"},
		{"config", "user.email", "dev@kubebuilder.test"},
		{"config", "user.name", "Kubebuilder Dev"},
	}
	for _, args := range commands {
		if _, err := runGitCommand(dir, args...); err != nil {
			t.Fatalf("initializing repo (%v): %v", args, err)
		}
	}

	createCommit(t, dir, "tracked.txt", "initial")
	return dir
}

func createCommit(t *testing.T, repo, file, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(repo, file), []byte(content+"\n"), 0o644); err != nil {
		t.Fatalf("writing file: %v", err)
	}
	if _, err := runGitCommand(repo, "add", file); err != nil {
		t.Fatalf("git add: %v", err)
	}
	commitEnv := append(os.Environ(),
		"GIT_COMMITTER_DATE=2006-01-02T15:04:05Z",
		"GIT_AUTHOR_DATE=2006-01-02T15:04:05Z",
	)
	cmd := exec.Command("git", "commit", "-m", fmt.Sprintf("commit %s", file))
	cmd.Dir = repo
	cmd.Env = append(commitEnv, "LC_ALL=C", "LANG=C")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, output)
	}
}
