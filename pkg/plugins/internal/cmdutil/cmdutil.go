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

package cmdutil

import (
	"github.com/spf13/afero"
)

// Scaffolder interface creates files to set up a controller manager
type Scaffolder interface {
	InjectFS(afero.Fs)
	// Scaffold performs the scaffolding
	Scaffold() error
}

// RunOptions represent the types used to implement the different commands
type RunOptions interface {
	// - Step 1: verify that the command can be run (e.g., go version, project version, arguments, ...).
	Validate() error
	// - Step 2: create the Scaffolder instance.
	GetScaffolder() (Scaffolder, error)
	// - Step 3: inject the filesystem into the Scaffolder instance. Doesn't need any method.
	// - Step 4: call the Scaffold method of the Scaffolder instance. Doesn't need any method.
	// - Step 5: finish the command execution.
	PostScaffold() error
}

// Run executes a command
func Run(options RunOptions, fs afero.Fs) error {
	// Step 1: validate
	if err := options.Validate(); err != nil {
		return err
	}

	// Step 2: get scaffolder
	scaffolder, err := options.GetScaffolder()
	if err != nil {
		return err
	}
	// Step 3: inject filesystem
	scaffolder.InjectFS(fs)
	// Step 4: scaffold
	if scaffolder != nil {
		if err := scaffolder.Scaffold(); err != nil {
			return err
		}
	}
	// Step 5: finish
	if err := options.PostScaffold(); err != nil {
		return err
	}

	return nil
}
