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

package file

import (
	"fmt"
	"path/filepath"
)

const prefix = "+kubebuilder:scaffold:"

var commentsByExt = map[string]string{
	// TODO(v3): machine-readable comments should not have spaces by Go convention. However,
	//  this is a backwards incompatible change, and thus should be done for next project version.
	".go":   "// ",
	".yaml": "# ",
	// When adding additional file extensions, update also the NewMarkerFor documentation and error
}

// Marker represents a machine-readable comment that will be used for scaffolding purposes
type Marker struct {
	comment string
	value   string
}

// NewMarkerFor creates a new marker customized for the specific file
// Supported file extensions: .go, .ext
func NewMarkerFor(path string, value string) Marker {
	ext := filepath.Ext(path)
	if comment, found := commentsByExt[ext]; found {
		return Marker{comment, value}
	}

	panic(fmt.Errorf("unknown file extension: '%s', expected '.go' or '.yaml'", ext))
}

// String implements Stringer
func (m Marker) String() string {
	return m.comment + prefix + m.value
}

// CodeFragments represents a set of code fragments
// A code fragment is a piece of code provided as a Go string, it may have multiple lines
type CodeFragments []string

// CodeFragmentsMap binds Markers and CodeFragments together
type CodeFragmentsMap map[Marker]CodeFragments
