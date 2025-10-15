/*
Copyright 2017 The Kubernetes Authors.

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
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	unknown                      = "unknown"
	develVersion                 = "(devel)"
	pseudoVersionTimestampLayout = "20060102150405"
)

// var needs to be used instead of const as ldflags is used to fill this
// information in the release process
var (
	kubeBuilderVersion      = unknown
	kubernetesVendorVersion = "1.34.1"
	goos                    = unknown
	goarch                  = unknown
	gitCommit               = "$Format:%H$" // sha1 from git, output of $(git rev-parse HEAD)

	buildDate = "1970-01-01T00:00:00Z" // build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
)

// version contains all the information related to the CLI version
type version struct {
	KubeBuilderVersion string `json:"kubeBuilderVersion"`
	KubernetesVendor   string `json:"kubernetesVendor"`
	GitCommit          string `json:"gitCommit"`
	BuildDate          string `json:"buildDate"`
	GoOs               string `json:"goOs"`
	GoArch             string `json:"goArch"`
}

// versionString returns the Full CLI version
func versionString() string {
	kubeBuilderVersion = getKubebuilderVersion()

	return fmt.Sprintf("Version: %#v", version{
		kubeBuilderVersion,
		kubernetesVendorVersion,
		gitCommit,
		buildDate,
		goos,
		goarch,
	})
}

// getKubebuilderVersion returns only the CLI version string
func getKubebuilderVersion() string {
	if strings.Contains(kubeBuilderVersion, "dirty") {
		return develVersion
	}
	if shouldResolveVersion(kubeBuilderVersion) {
		kubeBuilderVersion = resolveKubebuilderVersion()
	}
	return kubeBuilderVersion
}

func shouldResolveVersion(v string) bool {
	return v == "" || v == unknown || v == develVersion
}

func resolveKubebuilderVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Sum == "" {
			return develVersion
		}
		mainVersion := strings.TrimSpace(info.Main.Version)
		if mainVersion != "" && mainVersion != develVersion {
			return mainVersion
		}

		if v := pseudoVersionFromGit(info.Main.Path); v != "" {
			return v
		}
	}

	if v := pseudoVersionFromGit(""); v != "" {
		return v
	}

	return unknown
}

func pseudoVersionFromGit(modulePath string) string {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return ""
	}
	return pseudoVersionFromGitDir(modulePath, repoRoot)
}

func pseudoVersionFromGitDir(modulePath, repoRoot string) string {
	dirty, err := repoDirty(repoRoot)
	if err != nil {
		return ""
	}
	if dirty {
		return develVersion
	}

	commitHash, err := runGitCommand(repoRoot, "rev-parse", "--short=12", "HEAD")
	if err != nil || commitHash == "" {
		return ""
	}

	commitTimestamp, err := runGitCommand(repoRoot, "show", "-s", "--format=%ct", "HEAD")
	if err != nil || commitTimestamp == "" {
		return ""
	}
	seconds, err := strconv.ParseInt(commitTimestamp, 10, 64)
	if err != nil {
		return ""
	}
	timestamp := time.Unix(seconds, 0).UTC().Format(pseudoVersionTimestampLayout)

	if tag, err := runGitCommand(repoRoot, "describe", "--tags", "--exact-match"); err == nil {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			return tag
		}
	}

	if baseTag, err := runGitCommand(repoRoot, "describe", "--tags", "--abbrev=0"); err == nil {
		baseTag = strings.TrimSpace(baseTag)
		if semver.IsValid(baseTag) {
			if next := incrementPatch(baseTag); next != "" {
				return fmt.Sprintf("%s-0.%s-%s", next, timestamp, commitHash)
			}
		}
		if baseTag != "" {
			return baseTag
		}
	}

	major := moduleMajorVersion(modulePath)
	return buildDefaultPseudoVersion(major, timestamp, commitHash)
}

func repoDirty(repoRoot string) (bool, error) {
	status, err := runGitCommand(repoRoot, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		return false, err
	}
	return status != "", nil
}

func incrementPatch(tag string) string {
	trimmed := strings.TrimPrefix(tag, "v")
	trimmed = strings.SplitN(trimmed, "-", 2)[0]
	parts := strings.Split(trimmed, ".")
	if len(parts) < 3 {
		return ""
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return ""
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return ""
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return ""
	}
	patch++
	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}

func buildDefaultPseudoVersion(major int, timestamp, commitHash string) string {
	if major < 0 {
		major = 0
	}
	return fmt.Sprintf("v%d.0.0-%s-%s", major, timestamp, commitHash)
}

func moduleMajorVersion(modulePath string) int {
	if modulePath == "" {
		return 0
	}
	lastSlash := strings.LastIndex(modulePath, "/v")
	if lastSlash == -1 || lastSlash == len(modulePath)-2 {
		return 0
	}
	majorStr := modulePath[lastSlash+2:]
	if strings.Contains(majorStr, "/") {
		majorStr = majorStr[:strings.Index(majorStr, "/")]
	}
	major, err := strconv.Atoi(majorStr)
	if err != nil {
		return 0
	}
	return major
}

func findRepoRoot() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to determine caller")
	}

	if !filepath.IsAbs(currentFile) {
		abs, err := filepath.Abs(currentFile)
		if err != nil {
			return "", fmt.Errorf("getting absolute path: %w", err)
		}
		currentFile = abs
	}

	dir := filepath.Dir(currentFile)
	for {
		if dir == "" || dir == filepath.Dir(dir) {
			return "", fmt.Errorf("git repository root not found from %s", currentFile)
		}

		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		dir = filepath.Dir(dir)
	}
}

func runGitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("running git %v: %w", args, err)
	}
	return strings.TrimSpace(string(output)), nil
}
