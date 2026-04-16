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

package kustomize

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// CRDWriter writes CRDs to the CRD sub-chart's templates directory
type CRDWriter struct {
	outputDir string
}

// NewCRDWriter creates a new CRD writer
func NewCRDWriter(outputDir string) *CRDWriter {
	return &CRDWriter{
		outputDir: outputDir,
	}
}

// WriteCRDs writes all CRDs to the CRD sub-chart's templates directory
func (w *CRDWriter) WriteCRDs(fs machinery.Filesystem, crds []*unstructured.Unstructured) error {
	for _, crd := range crds {
		if crd == nil {
			continue
		}

		// Extract CRD name for filename
		crdName := crd.GetName()
		if crdName == "" {
			return fmt.Errorf("CRD missing name")
		}

		// Generate filename from CRD name (e.g., memcacheds.cache.example.com -> memcacheds.yaml)
		// Use the plural name before the first dot
		parts := strings.Split(crdName, ".")
		filename := parts[0] + ".yaml"

		// Convert to YAML
		yamlBytes, err := yaml.Marshal(crd.Object)
		if err != nil {
			return fmt.Errorf("failed to marshal CRD %s to YAML: %w", crdName, err)
		}

		// Write to crds/templates/ directory
		filePath := filepath.Join(w.outputDir, "chart", "crds", "templates", filename)

		// Create directory if it doesn't exist
		dir := filepath.Dir(filePath)
		if err := fs.FS.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if err := afero.WriteFile(fs.FS, filePath, yamlBytes, 0o644); err != nil {
			return fmt.Errorf("failed to write CRD %s to %s: %w", crdName, filePath, err)
		}
	}

	return nil
}
