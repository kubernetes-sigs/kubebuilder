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

package scaffolds

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
)

// ensureDockerfileHasCopy ensures the Dockerfile contains a "COPY src/ dst/" line.
// If absent, it inserts the line right after the *last* COPY instruction (case-insensitive).
func ensureDockerfileHasCopy(dockerfilePath, src, dst, label string) error {
	// Ensure trailing slashes to copy directory contents consistently
	if !strings.HasSuffix(src, "/") {
		src += "/"
	}
	if !strings.HasSuffix(dst, "/") {
		dst += "/"
	}
	copyStmt := fmt.Sprintf("COPY %s %s", src, dst)

	// Check if Dockerfile exists
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		slog.Warn("Dockerfile not found, skipping COPY insertion", "path", dockerfilePath, "what", label)
		return nil
	}

	// Idempotency check
	has, err := util.HasFileContentWith(dockerfilePath, copyStmt)
	if err != nil {
		slog.Warn("Could not check Dockerfile for COPY", "what", label, "error", err)
		return fmt.Errorf("unable to find COPY: %s", err)
	}
	if has {
		slog.Debug("COPY already exists in Dockerfile", "what", label, "copy", copyStmt)
		return fmt.Errorf("COPY already exist: %s", err)
	}

	// Insert after the last COPY (case-sensitive; matches "COPY ", including "COPY --chown=...")
	if err := util.InsertAfterLastMatchString(dockerfilePath, "COPY ", copyStmt); err != nil {
		slog.Warn(
			"Could not ensure Dockerfile has the required COPY. Please add it manually.",
			"what", label,
			"copy", copyStmt,
			"error", err,
		)
		return fmt.Errorf("enable to insert COPY: %w", err)
	}
	slog.Info(fmt.Sprintf("Added %s to Dockerfile", copyStmt))
	return nil
}
