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

package machinery

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"strings"
)

// FeatureGateMarker represents a feature gate marker found in source code
type FeatureGateMarker struct {
	GateName string
	Line     int
	File     string
}

// FeatureGateMarkerParser parses Go source files for feature gate markers
type FeatureGateMarkerParser struct {
	// Regex to match feature gate markers
	// Matches: // +feature-gate gate-name
	markerRegex *regexp.Regexp
}

// NewFeatureGateMarkerParser creates a new feature gate marker parser
func NewFeatureGateMarkerParser() *FeatureGateMarkerParser {
	return &FeatureGateMarkerParser{
		markerRegex: regexp.MustCompile(`^\s*//\s*\+feature-gate\s+([a-zA-Z0-9_-]+)\s*$`),
	}
}

// ParseFile parses a Go source file for feature gate markers
func (p *FeatureGateMarkerParser) ParseFile(filePath string) ([]FeatureGateMarker, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	var markers []FeatureGateMarker
	scanner := bufio.NewScanner(file)
	lineNum := 0
	var lines []string

	// First pass: collect all lines
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	// Second pass: analyze lines to find feature gate markers associated with real code
	for i, line := range lines {
		lineNum = i + 1

		matches := p.markerRegex.FindStringSubmatch(line)
		if len(matches) == 2 {
			// Check if this marker is associated with actual code (not just comments)
			if p.isMarkerAssociatedWithCode(lines, i) {
				markers = append(markers, FeatureGateMarker{
					GateName: matches[1],
					Line:     lineNum,
					File:     filePath,
				})
			}
		}
	}

	return markers, nil
}

// isMarkerAssociatedWithCode checks if a feature gate marker is associated with actual code
// rather than just commented code
func (p *FeatureGateMarkerParser) isMarkerAssociatedWithCode(lines []string, markerLineIndex int) bool {
	// Look ahead for actual code (not comments) within the next few lines
	// Feature gate markers should be followed by actual field declarations
	for i := markerLineIndex + 1; i < len(lines) && i <= markerLineIndex+5; i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// If we hit another marker, stop looking
		if strings.Contains(trimmed, "+feature-gate") {
			break
		}

		// If we hit end of struct, stop looking
		if strings.Contains(trimmed, "}") {
			break
		}

		// Skip pure comment lines
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Check if this line contains actual Go code (field declaration, struct, etc.)
		// Look for patterns like: FieldName Type `json:"fieldName,omitempty"`
		if strings.Contains(trimmed, "`json:") ||
			strings.Contains(trimmed, "string") ||
			strings.Contains(trimmed, "int") ||
			strings.Contains(trimmed, "bool") ||
			strings.Contains(trimmed, "*") {
			return true
		}
	}

	return false
}

// ParseDirectory parses all Go files in a directory for feature gate markers
func (p *FeatureGateMarkerParser) ParseDirectory(dirPath string) ([]FeatureGateMarker, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dirPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory %s: %w", dirPath, err)
	}

	var allMarkers []FeatureGateMarker

	for _, pkg := range pkgs {
		for fileName, file := range pkg.Files {
			markers := p.parseASTFile(fileName, file, fset)
			allMarkers = append(allMarkers, markers...)
		}
	}

	return allMarkers, nil
}

// parseASTFile parses an AST file for feature gate markers
func (p *FeatureGateMarkerParser) parseASTFile(fileName string, file *ast.File, fset *token.FileSet) []FeatureGateMarker {
	var markers []FeatureGateMarker

	// Parse comments for feature gate markers
	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			matches := p.markerRegex.FindStringSubmatch(comment.Text)
			if len(matches) == 2 {
				// Check if this marker is associated with actual code (not just comments)
				if p.isASTMarkerAssociatedWithCode(file, fset, comment) {
					markers = append(markers, FeatureGateMarker{
						GateName: matches[1],
						Line:     fset.Position(comment.Pos()).Line,
						File:     fileName,
					})
				}
			}
		}
	}

	return markers
}

// isASTMarkerAssociatedWithCode checks if a feature gate marker in AST is associated with actual code
// rather than just commented code
func (p *FeatureGateMarkerParser) isASTMarkerAssociatedWithCode(file *ast.File, fset *token.FileSet, comment *ast.Comment) bool {
	commentPos := fset.Position(comment.Pos())

	// Look for AST nodes that come after this comment
	for _, decl := range file.Decls {
		declPos := fset.Position(decl.Pos())

		// Look for field declarations within structs
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if structType, ok := typeSpec.Type.(*ast.StructType); ok {
						for _, field := range structType.Fields.List {
							fieldPos := fset.Position(field.Pos())
							// Check if this field is within a reasonable distance from the comment (before or after)
							if fieldPos.Line >= commentPos.Line-2 && fieldPos.Line <= commentPos.Line+5 {
								// This field is close to the comment, so the marker is likely associated with it
								return true
							}
						}
					}
				}
			}
		}

		// If we've looked too far ahead, stop
		if declPos.Line > commentPos.Line+10 {
			break
		}
	}

	return false
}

// ExtractFeatureGates extracts all unique feature gate names from markers
func ExtractFeatureGates(markers []FeatureGateMarker) []string {
	gateMap := make(map[string]bool)
	for _, marker := range markers {
		gateMap[marker.GateName] = true
	}

	var gates []string
	for gate := range gateMap {
		gates = append(gates, gate)
	}
	return gates
}

// ValidateFeatureGates validates that all required feature gates are enabled
func ValidateFeatureGates(requiredGates []string, enabledGates FeatureGates) []string {
	var missing []string
	for _, gate := range requiredGates {
		if !enabledGates.IsEnabled(gate) {
			missing = append(missing, gate)
		}
	}
	return missing
}
