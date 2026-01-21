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

// Go module sanity checker validates module compatibility before release:
// 1. Validates file paths with x/mod/module.CheckFilePath
// 2. Ensures required retracted versions are present in go.mod
// 3. Reads module path and Go version from go.mod
// 4. Creates a consumer module to test installability
// 5. Runs `go mod tidy` and `go build ./...` to verify module works
//
// This prevents releasing tags that break `go install`.
//
// Run with:
//   go run ./hack/test/check_go_module.go

package main

import (
	"bufio"
	"bytes"
	"fmt"
	log "log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/module"
)

func main() {
	if err := checkFilePaths(); err != nil {
		log.Error("file path validation failed", "error", err)
		os.Exit(1)
	}

	if err := checkRetractedVersions(); err != nil {
		log.Error("retracted version check failed", "error", err)
		os.Exit(1)
	}

	modulePath, goVersion, err := readGoModInfo()
	if err != nil {
		log.Error("failed to read go.mod", "error", err)
		os.Exit(1)
	}

	if err := setupAndCheckConsumer(modulePath, goVersion); err != nil {
		log.Error("consumer module validation failed", "error", err)
		os.Exit(1)
	}

	log.Info("Go module compatibility check passed")
}

func checkFilePaths() error {
	log.Info("Checking Go module file paths")

	out, err := exec.Command("git", "ls-files").Output()
	if err != nil {
		return fmt.Errorf("failed to list git tracked files: %w", err)
	}

	var invalidPaths []string
	for line := range strings.SplitSeq(string(out), "\n") {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}

		if err := module.CheckFilePath(path); err != nil {
			invalidPaths = append(invalidPaths, fmt.Sprintf("  %q: %v", path, err))
		}
	}

	if len(invalidPaths) > 0 {
		var buf bytes.Buffer
		buf.WriteString("invalid file paths found:\n")
		for _, p := range invalidPaths {
			buf.WriteString(p)
			buf.WriteByte('\n')
		}
		return fmt.Errorf("%s", buf.String())
	}

	log.Info("File path validation passed")
	return nil
}

func checkRetractedVersions() error {
	log.Info("Checking for required retracted versions in go.mod")

	content, err := os.ReadFile("go.mod")
	if err != nil {
		return fmt.Errorf("failed to read go.mod: %w", err)
	}

	requiredRetractions := []string{
		"retract v4.10.0", // invalid filename causes go get/install failure (#5211)
	}

	for _, retract := range requiredRetractions {
		if !strings.Contains(string(content), retract) {
			return fmt.Errorf("missing required retraction: %s", retract)
		}
	}

	log.Info("Retracted versions check passed")
	return nil
}

func readGoModInfo() (modulePath, goVersion string, err error) {
	log.Info("Reading module info from go.mod")

	f, openErr := os.Open("go.mod")
	if openErr != nil {
		return "", "", fmt.Errorf("failed to open go.mod: %w", openErr)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			log.Warn("failed to close go.mod", "error", closeErr)
		}
	}()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())

		// Read module path from first line
		if after, ok := strings.CutPrefix(line, "module "); ok {
			modulePath = strings.TrimSpace(after)
			log.Info("Found module path", "module", modulePath)
		}

		// Read Go version
		if after, ok := strings.CutPrefix(line, "go "); ok {
			goVersion = strings.TrimSpace(after)
			log.Info("Found Go version", "version", goVersion)
		}

		// Stop once we have both
		if modulePath != "" && goVersion != "" {
			break
		}
	}

	if modulePath == "" {
		return "", "", fmt.Errorf("no 'module' directive found in go.mod")
	}
	if goVersion == "" {
		return "", "", fmt.Errorf("no 'go' directive found in go.mod")
	}

	return modulePath, goVersion, nil
}

func setupAndCheckConsumer(modulePath, goVersion string) error {
	log.Info("Creating consumer module", "module", modulePath, "go_version", goVersion)

	// Create temporary directory under hack/test/ (covered by **/e2e-*/** in .gitignore)
	consumerDir := filepath.Join("hack", "test", "e2e-module-check")
	if err := os.MkdirAll(consumerDir, 0o755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(consumerDir); err != nil {
			log.Warn("failed to cleanup temp dir", "dir", consumerDir, "error", err)
		}
	}()

	if err := writeConsumerFiles(consumerDir, modulePath, goVersion); err != nil {
		return err
	}

	log.Info("Running go mod tidy in consumer module")
	if err := runCommand(consumerDir, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	log.Info("Building consumer module")
	if err := runCommand(consumerDir, "go", "build", "./..."); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	log.Info("Consumer module build succeeded")
	return nil
}

func writeConsumerFiles(consumerDir, modulePath, goVersion string) error {
	goMod := fmt.Sprintf(`module module-consumer

go %s

require %s v4.0.0-00010101000000-000000000000

replace %s => ../../..
`, goVersion, modulePath, modulePath)

	// Use a basic import from the module to verify it can be consumed
	mainGo := fmt.Sprintf(`package main

import (
	_ "%s/pkg/plugins/golang/v4"
)

func main() {}
`, modulePath)

	if err := os.WriteFile(filepath.Join(consumerDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		return fmt.Errorf("failed to write consumer go.mod: %w", err)
	}

	if err := os.WriteFile(filepath.Join(consumerDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		return fmt.Errorf("failed to write consumer main.go: %w", err)
	}

	return nil
}

// runCommand executes a command in the specified directory with stdout/stderr connected
func runCommand(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command %s failed in %s: %w", name, dir, err)
	}
	return nil
}
