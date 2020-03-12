/*
Copyright 2020 The Kubernetes Authors.

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

package dependencies

import (
	"fmt"
)

// CmdParseError represents an error parsing the output of a command execution
type CmdParseError struct {
	cmd string
	err error
}

// Error implements error interface
func (e CmdParseError) Error() string {
	return fmt.Sprintf("unable to parse `%s`: %v", e.cmd, e.err)
}

// Unwrap implements Wrapper interface
func (e CmdParseError) Unwrap() error {
	return e.err
}

// RequiredVersionError represents an unfulfilled minimum version error
type RequiredVersionError struct {
	cmd     string
	version semanticVersion
	min     semanticVersion
}

// Error implements error interface
func (e RequiredVersionError) Error() string {
	return fmt.Sprintf("requires %s version (%v) >= %v", e.cmd, e.version, e.min)
}

// CompilationVersionMatchError represents a mismatch between the version used to compile kubebuilder
// and the version available at the path that will be used to compile the scaffolded project
type CompilationVersionMatchError struct {
	info goBuildInfo
}

// Error implements error interface
func (e CompilationVersionMatchError) Error() string {
	return fmt.Sprintf("go version used to compile kubebuilder (%v) and current go version (%v) do not match",
		compilationGoInfo.version, e.info.version)
}
