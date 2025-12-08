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

package helpers

import (
	"bufio"
	"bytes"
	"io/fs"
	log "log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type ConflictSummary struct {
	Makefile bool // Makefile or makefile conflicted
	API      bool // anything under api/ or apis/ conflicted
	AnyGo    bool // any *.go file anywhere conflicted
}

// ConflictResult provides detailed conflict information for multiple use cases
type ConflictResult struct {
	Summary        ConflictSummary
	SourceFiles    []string // conflicted source files
	GeneratedFiles []string // conflicted generated files
}

// isGeneratedKB returns true for Kubebuilder-generated artifacts.
// Moved from open_gh_issue.go to avoid duplication
func isGeneratedKB(path string) bool {
	return strings.Contains(path, "/zz_generated.") ||
		strings.HasPrefix(path, "config/crd/bases/") ||
		strings.HasPrefix(path, "config/rbac/") ||
		path == "dist/install.yaml" ||
		// Generated deepcopy files
		strings.HasSuffix(path, "_deepcopy.go")
}

// FindConflictFiles performs unified conflict detection for both conflict handling and GitHub issue generation
func FindConflictFiles() ConflictResult {
	result := ConflictResult{
		SourceFiles:    []string{},
		GeneratedFiles: []string{},
	}

	// Use git index for fast conflict detection first
	gitConflicts := getGitIndexConflicts()

	// Filesystem scan for conflict markers
	fsConflicts := scanFilesystemForConflicts()

	// Combine results and categorize
	allConflicts := make(map[string]bool)
	for _, f := range gitConflicts {
		allConflicts[f] = true
	}
	for _, f := range fsConflicts {
		allConflicts[f] = true
	}

	// Categorize into source vs generated
	for file := range allConflicts {
		if isGeneratedKB(file) {
			result.GeneratedFiles = append(result.GeneratedFiles, file)
		} else {
			result.SourceFiles = append(result.SourceFiles, file)
		}
	}

	sort.Strings(result.SourceFiles)
	sort.Strings(result.GeneratedFiles)

	// Build summary for existing conflict.go usage
	result.Summary = ConflictSummary{
		Makefile: hasConflictInFiles(allConflicts, "Makefile", "makefile"),
		API:      hasConflictInPaths(allConflicts, "api", "apis"),
		AnyGo:    hasGoConflictInFiles(allConflicts),
	}

	return result
}

// DetectConflicts maintains backward compatibility
func DetectConflicts() ConflictSummary {
	return FindConflictFiles().Summary
}

// getGitIndexConflicts uses git ls-files to quickly find unmerged entries
func getGitIndexConflicts() []string {
	out, err := exec.Command("git", "ls-files", "-u").Output()
	if err != nil {
		return nil
	}

	conflicts := make(map[string]bool)
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			file := strings.Join(fields[3:], " ")
			conflicts[file] = true
		}
	}

	result := make([]string, 0, len(conflicts))
	for file := range conflicts {
		result = append(result, file)
	}
	return result
}

// scanFilesystemForConflicts scans the working directory for conflict markers
func scanFilesystemForConflicts() []string {
	type void struct{}
	skipDir := map[string]void{
		".git":   {},
		"vendor": {},
		"bin":    {},
	}

	const maxBytes = 2 << 20 // 2 MiB per file

	markersPrefix := [][]byte{
		[]byte("<<<<<<< "),
		[]byte(">>>>>>> "),
	}
	markerExact := []byte("=======")

	var conflicts []string

	_ = filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // best-effort
		}
		// Skip unwanted directories
		if d.IsDir() {
			if _, ok := skipDir[d.Name()]; ok {
				return filepath.SkipDir
			}
			return nil
		}

		// Quick size check
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		if fi.Size() > maxBytes {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				log.Warn("failed to close file", "path", path, "error", cerr)
			}
		}()

		found := false
		sc := bufio.NewScanner(f)
		// allow long lines (YAML/JSON)
		buf := make([]byte, 0, 1024*1024)
		sc.Buffer(buf, 4<<20)

		for sc.Scan() {
			b := sc.Bytes()
			// starts with conflict markers
			for _, p := range markersPrefix {
				if bytes.HasPrefix(b, p) {
					found = true
					break
				}
			}
			// exact middle marker line
			if !found && bytes.Equal(b, markerExact) {
				found = true
			}
			if found {
				break
			}
		}

		if found {
			conflicts = append(conflicts, path)
		}
		return nil
	})

	return conflicts
}

// Helper functions for backward compatibility
func hasConflictInFiles(conflicts map[string]bool, paths ...string) bool {
	for _, path := range paths {
		if conflicts[path] {
			return true
		}
	}
	return false
}

func hasConflictInPaths(conflicts map[string]bool, pathPrefixes ...string) bool {
	for file := range conflicts {
		for _, prefix := range pathPrefixes {
			if strings.HasPrefix(file, prefix+"/") || file == prefix {
				return true
			}
		}
	}
	return false
}

func hasGoConflictInFiles(conflicts map[string]bool) bool {
	for file := range conflicts {
		if strings.HasSuffix(file, ".go") {
			return true
		}
	}
	return false
}

// DecideMakeTargets applies simple policy over the summary.
func DecideMakeTargets(cs ConflictSummary) []string {
	all := []string{"manifests", "generate", "fmt", "vet", "lint-fix"}
	if cs.Makefile {
		return nil
	}
	keep := make([]string, 0, len(all))
	for _, t := range all {
		if cs.API && (t == "manifests" || t == "generate") {
			continue
		}
		if cs.AnyGo && (t == "fmt" || t == "vet" || t == "lint-fix") {
			continue
		}
		keep = append(keep, t)
	}
	return keep
}
