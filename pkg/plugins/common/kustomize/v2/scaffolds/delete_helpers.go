/*
Copyright 2026 The Kubernetes Authors.

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

package scaffolds

import (
	"bufio"
	"bytes"
	"fmt"
	log "log/slog"

	"github.com/spf13/afero"
)

// removeFileIfExists removes a file if it exists (best effort helper)
// Used by delete scaffolders to clean up files
func removeFileIfExists(afs afero.Fs, path string) error {
	exists, err := afero.Exists(afs, path)
	if err != nil {
		return fmt.Errorf("failed to check file: %w", err)
	}
	if !exists {
		return nil // Not an error if file doesn't exist
	}

	if err := afs.Remove(path); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	log.Info("Deleted file", "file", path)
	return nil
}

// removeLinesFromKustomization removes specific lines from a kustomization file
// Returns true if any lines were found and removed, false if none found
func removeLinesFromKustomization(afs afero.Fs, filePath string, linesToRemove []string) (bool, error) {
	exists, err := afero.Exists(afs, filePath)
	if err != nil {
		return false, fmt.Errorf("failed to check file: %w", err)
	}
	if !exists {
		return false, nil
	}

	content, err := afero.ReadFile(afs, filePath)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	// Build set of exact lines to remove (preserving indentation)
	removeSet := make(map[string]bool)
	for _, line := range linesToRemove {
		removeSet[line] = true
	}

	// Read file line by line and skip target lines
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var out bytes.Buffer
	removed := false

	for scanner.Scan() {
		line := scanner.Text()
		// Compare exact line (including indentation)
		if removeSet[line] {
			removed = true
			continue // Skip this line
		}
		out.WriteString(line)
		out.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("failed to scan file: %w", err)
	}

	if !removed {
		return false, nil
	}

	// Write the modified content back
	if err := afero.WriteFile(afs, filePath, out.Bytes(), 0o644); err != nil {
		return false, fmt.Errorf("failed to write file: %w", err)
	}

	return true, nil
}
