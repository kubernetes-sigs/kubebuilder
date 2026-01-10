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
