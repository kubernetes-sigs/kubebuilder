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
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	errFoundConflict = errors.New("found-conflict")
	errGoConflict    = errors.New("go-conflict")
)

type ConflictSummary struct {
	Makefile bool // Makefile or makefile conflicted
	API      bool // anything under api/ or apis/ conflicted
	AnyGo    bool // any *.go file anywhere conflicted
}

func DetectConflicts() ConflictSummary {
	return ConflictSummary{
		Makefile: hasConflict("Makefile", "makefile"),
		API:      hasConflict("api", "apis"),
		AnyGo:    hasGoConflicts(), // checks all *.go in repo (index fast path + FS scan)
	}
}

// hasConflict: file/dir conflicts via index fast path + marker scan.
func hasConflict(paths ...string) bool {
	if len(paths) == 0 {
		return false
	}
	// Fast path: any unmerged entry under these pathspecs?
	args := append([]string{"ls-files", "-u", "--"}, paths...)
	out, err := exec.Command("git", args...).Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return true
	}

	// Fallback: scan for conflict markers.
	hasMarkers := func(p string) bool {
		// Best-effort, skip large likely-binaries.
		if fi, err := os.Stat(p); err == nil && fi.Size() > 1<<20 {
			return false
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return false
		}
		s := string(b)
		return strings.Contains(s, "<<<<<<<") &&
			strings.Contains(s, "=======") &&
			strings.Contains(s, ">>>>>>>")
	}

	for _, root := range paths {
		info, err := os.Stat(root)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			if hasMarkers(root) {
				return true
			}
			continue
		}

		werr := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil
			}
			// Skip obvious noise dirs.
			if d.Name() == ".git" || strings.Contains(p, string(filepath.Separator)+".git"+string(filepath.Separator)) {
				return nil
			}
			if hasMarkers(p) {
				return errFoundConflict
			}
			return nil
		})
		if errors.Is(werr, errFoundConflict) {
			return true
		}
	}
	return false
}

// hasGoConflicts: any *.go file conflicted (repo-wide).
func hasGoConflicts(roots ...string) bool {
	// Fast path: any unmerged *.go anywhere?
	if out, err := exec.Command("git", "ls-files", "-u", "--", "*.go").Output(); err == nil {
		if len(strings.TrimSpace(string(out))) > 0 {
			return true
		}
	}
	// Fallback: filesystem scan (repo-wide or limited to roots if provided).
	if len(roots) == 0 {
		roots = []string{"."}
	}
	for _, root := range roots {
		werr := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() || !strings.HasSuffix(p, ".go") {
				return nil
			}
			// Skip .git and large files.
			if strings.Contains(p, string(filepath.Separator)+".git"+string(filepath.Separator)) {
				return nil
			}
			if fi, err := os.Stat(p); err == nil && fi.Size() > 1<<20 {
				return nil
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return nil
			}
			s := string(b)
			if strings.Contains(s, "<<<<<<<") &&
				strings.Contains(s, "=======") &&
				strings.Contains(s, ">>>>>>>") {
				return errGoConflict
			}
			return nil
		})
		if errors.Is(werr, errGoConflict) {
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
