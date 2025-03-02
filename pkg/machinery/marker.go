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
	"strings"
)

const kbPrefix = "+kubebuilder:scaffold:"

var commentsByExt = map[string]string{
	".go":   "//",
	".yaml": "#",
	".yml":  "#",
	".rs":   "//",
	// When adding additional file extensions, update also the NewMarkerFor documentation and error
}

// Marker represents a machine-readable comment that will be used for scaffolding purposes
type Marker struct {
	prefix  string
	comment string
	value   string
}

// NewMarkerFor creates a new marker customized for the specific file. The created marker
// is prefixed with `+kubebuilder:scaffold:` the default prefix for kubebuilder.
// Supported file extensions: .go, .yaml, .yml.
func NewMarkerFor(path string, value string) Marker {
	return NewMarkerWithPrefixFor(kbPrefix, path, value)
}

// NewMarkerWithPrefixFor creates a new custom prefixed marker customized for the specific file
// Supported file extensions: .go, .yaml, .yml
func NewMarkerWithPrefixFor(prefix string, path string, value string) Marker {
	ext := filepath.Ext(path)
	if comment, found := commentsByExt[ext]; found {
		return Marker{
			prefix:  markerPrefix(prefix),
			comment: comment,
			value:   value,
		}
	}

	extensions := make([]string, 0, len(commentsByExt))
	for extension := range commentsByExt {
		extensions = append(extensions, fmt.Sprintf("%q", extension))
	}
	panic(fmt.Errorf("unknown file extension: '%s', expected one of: %s", ext, strings.Join(extensions, ", ")))
}

// String implements Stringer
func (m Marker) String() string {
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
