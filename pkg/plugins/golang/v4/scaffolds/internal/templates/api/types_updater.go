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

package api

import (
	"bytes"
	"fmt"
	log "log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

const (
	storageVersionMarker = "\n// +kubebuilder:storageversion"
)

var _ machinery.Template = &TypesUpdater{}

// TypesUpdater updates an existing API types file to add conversion-related markers
type TypesUpdater struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.ResourceMixin
}

// GetPath implements file.Builder
func (f *TypesUpdater) GetPath() string {
	if f.MultiGroup && f.Resource.Group != "" {
		f.Path = filepath.Join("api", "%[group]", "%[version]", "%[kind]_types.go")
	} else {
		f.Path = filepath.Join("api", "%[version]", "%[kind]_types.go")
	}

	return f.Resource.Replacer().Replace(f.Path)
}

// GetIfExistsAction implements file.Builder
func (*TypesUpdater) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.OverwriteFile
}

// SetTemplateDefaults implements file.Template
func (f *TypesUpdater) SetTemplateDefaults() error {
	filePath := f.GetPath()

	// Read the existing file
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Error("failed to read types file", "file", filePath, "error", err)
		return fmt.Errorf("failed to read types file: %w", err)
	}

	fileContent := string(content)
	modified := false

	// Check if we need to add storage version marker for conversion webhooks
	if f.Resource.HasConversionWebhook() && !bytes.Contains(content, []byte("+kubebuilder:storageversion")) {
		fileContent = f.addStorageVersionMarker(fileContent)
		modified = true
	}

	if !modified {
		// No updates needed, skip writing
		return nil
	}

	f.TemplateBody = fileContent
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

// addStorageVersionMarker adds the storage version marker after +kubebuilder:object:root=true
func (f *TypesUpdater) addStorageVersionMarker(content string) string {
	// Try to match the specific Kind's type definition (handles multigroup with multiple types)
	typePatternStr := fmt.Sprintf(
		`(?m)^(//\s*\+kubebuilder:object:root=true)\s*$(?:\s*//.*$)*\s*type\s+%s\s+struct`,
		f.Resource.Kind)
	typePattern := regexp.MustCompile(typePatternStr)

	if match := typePattern.FindStringSubmatch(content); len(match) > 1 {
		rootMarker := match[1]
		idx := strings.Index(content, rootMarker)
		if idx != -1 {
			insertPos := idx + len(rootMarker)
			return content[:insertPos] + storageVersionMarker + content[insertPos:]
		}
	}

	// Fallback: find first +kubebuilder:object:root=true marker
	simplePattern := regexp.MustCompile(`(?m)^(//\s*\+kubebuilder:object:root=true)\s*$`)
	if match := simplePattern.FindStringIndex(content); match != nil {
		log.Info("Adding storage version marker to first type definition",
			"kind", f.Resource.Kind)
		return content[:match[1]] + storageVersionMarker + content[match[1]:]
	}

	log.Warn("Could not find +kubebuilder:object:root=true marker",
		"kind", f.Resource.Kind,
		"suggestion", "Manually add // +kubebuilder:storageversion")

	return content
}
