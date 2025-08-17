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
	"fmt"
	log "log/slog"
	"os/exec"
)

// CommitIgnoreEmpty commits the staged changes with the provided message.
func CommitIgnoreEmpty(msg, ctx string) error {
	cmd := exec.Command("git", "commit", "--no-verify", "-m", msg)
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			// nothing to commit
			log.Info("No changes to commit", "context", ctx, "message", msg)
			return nil
		}
		return fmt.Errorf("git commit failed (%s): %w", ctx, err)
	}
	return nil
}

// CleanWorktree removes everything in the repo root except .git so the next
// checkout writes a verbatim snapshot of the source branch.
func CleanWorktree(label string) error {
	if err := exec.Command("sh", "-c",
		"find . -mindepth 1 -maxdepth 1 ! -name '.git' -exec rm -rf {} +").Run(); err != nil {
		return fmt.Errorf("cleanup for %s: %w", label, err)
	}
	return nil
}
