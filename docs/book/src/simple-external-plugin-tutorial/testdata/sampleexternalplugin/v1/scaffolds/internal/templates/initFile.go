/*
Copyright 2022 The Kubernetes Authors.

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
package templates

import "fmt"

// InitFile represents the InitFile.txt
type InitFile struct {
	Name     string
	Contents string
	domain   string
}

// InitFileOptions is a way to set configurable options for the Init file
type InitFileOptions func(inf *InitFile)

// WithDomain sets the number to be used in the resulting InitFile
func WithDomain(domain string) InitFileOptions {
	return func(inf *InitFile) {
		inf.domain = domain
	}
}

// NewInitFile returns a new InitFile with
func NewInitFile(opts ...InitFileOptions) *InitFile {
	initFile := &InitFile{
		Name: "initFile.txt",
	}

	for _, opt := range opts {
		opt(initFile)
	}

	initFile.Contents = fmt.Sprintf(initFileTemplate, initFile.domain)

	return initFile
}

const initFileTemplate = "A simple text file created with the `init` subcommand\nDOMAIN: %s"
