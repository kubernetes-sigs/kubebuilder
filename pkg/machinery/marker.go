/*
Copyright 2019 The Kubernetes Authors.

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
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

const (
	kbPrefix  = "+kubebuilder:scaffold:"
	goFileExt = ".go"
)

var commentsByExt = map[string]string{
	goFileExt: "//",
	".yaml":   "#",
	".yml":    "#",
	// When adding additional file extensions, update the "Supported file extensions" doc comment on
	// ParseMarkerFor and ParseMarkerWithPrefixFor.
}

// Marker represents a machine-readable comment that will be used for scaffolding purposes
type Marker struct {
	prefix  string
	comment string
	value   string
}

// must panics with err if non-nil, otherwise returns m.
func must(m Marker, err error) Marker {
	if err != nil {
		panic(err)
	}
	return m
}

// NewMarkerFor creates a new marker customized for the specific file. The created marker
// is prefixed with `+kubebuilder:scaffold:` the default prefix for kubebuilder.
// Panics on unsupported file extensions; use ParseMarkerFor when the path may come
// from untrusted input and an error is preferable to a panic.
func NewMarkerFor(path string, value string) Marker {
	return must(ParseMarkerFor(path, value))
}

// ParseMarkerFor creates a new marker customized for the specific file. The created marker
// is prefixed with `+kubebuilder:scaffold:` the default prefix for kubebuilder.
// Returns an error for unsupported file extensions instead of panicking.
// Supported file extensions: .go, .yaml, .yml.
func ParseMarkerFor(path string, value string) (Marker, error) {
	return ParseMarkerWithPrefixFor(kbPrefix, path, value)
}

// NewMarkerWithPrefixFor creates a new custom prefixed marker customized for the specific file.
// Panics on unsupported file extensions; use ParseMarkerWithPrefixFor when the path may come
// from untrusted input and an error is preferable to a panic.
func NewMarkerWithPrefixFor(prefix string, path string, value string) Marker {
	return must(ParseMarkerWithPrefixFor(prefix, path, value))
}

// ParseMarkerWithPrefixFor creates a new custom prefixed marker customized for the specific file.
// Returns an error for unsupported file extensions instead of panicking.
// Supported file extensions: .go, .yaml, .yml.
func ParseMarkerWithPrefixFor(prefix string, path string, value string) (Marker, error) {
	ext := filepath.Ext(path)
	if comment, found := commentsByExt[ext]; found {
		return Marker{
			prefix:  markerPrefix(prefix),
			comment: comment,
			value:   value,
		}, nil
	}

	exts := make([]string, 0, len(commentsByExt))
	for e := range commentsByExt {
		exts = append(exts, e)
	}
	slices.Sort(exts)
	for i, e := range exts {
		exts[i] = fmt.Sprintf("%q", e)
	}
	list := strings.Join(exts, ", ")

	// ext=="" covers plain extensionless names (e.g. Makefile).
	// ext=="." covers trailing-dot files (e.g. file.).
	if ext == "" || ext == "." {
		return Marker{}, fmt.Errorf("path %q has no file extension, expected one of: %s", path, list)
	}

	// Dotfiles like ".gitignore" land here, not above: filepath.Ext treats the leading dot as the
	// final dot, so Ext(".gitignore") == ".gitignore", not "". They're genuinely unknown extensions.
	return Marker{}, fmt.Errorf("path %q has unknown file extension %q, expected one of: %s", path, ext, list)
}

// String implements Stringer
func (m Marker) String() string {
	if m.comment == "" {
		return ""
	}
	return m.comment + " " + m.prefix + m.value
}

// EqualsLine compares a marker with a string representation to check if they are the same marker
func (m Marker) EqualsLine(line string) bool {
	line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), m.comment))
	return line == m.prefix+m.value
}

// CodeFragments represents a set of code fragments
// A code fragment is a piece of code provided as a Go string, it may have multiple lines
type CodeFragments []string

// CodeFragmentsMap binds Markers and CodeFragments together
type CodeFragmentsMap map[Marker]CodeFragments

func markerPrefix(prefix string) string {
	trimmed := strings.TrimSpace(prefix)
	var builder strings.Builder
	if !strings.HasPrefix(trimmed, "+") {
		builder.WriteString("+")
	}
	builder.WriteString(trimmed)
	if !strings.HasSuffix(trimmed, ":") {
		builder.WriteString(":")
	}

	return builder.String()
}
