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
	"sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
)

// RunOptions represent the types used to implement the different commands
type RunOptions interface {
	// The following steps define a generic logic to follow when developing new commands. Some steps may be no-ops.
	// - Step 1: load the config failing if expected but not found or if not expected but found
	LoadConfig() (*config.Config, error)
	// - Step 2: verify that the command can be run (e.g., go version, project version, arguments, ...)
	Validate(*config.Config) error
	// - Step 3: create the Scaffolder instance
	GetScaffolder(*config.Config) (scaffold.Scaffolder, error)
	// - Step 4: call the Scaffold method of the Scaffolder instance
	// Doesn't need any method
	// - Step 5: finish the command execution
	PostScaffold(*config.Config) error
}

// Run executes a command
func Run(options RunOptions) error {
	// Step 1: load config
	projectConfig, err := options.LoadConfig()
	if err != nil {
		return err
	}

	// Step 2: validate
	if err := options.Validate(projectConfig); err != nil {
		return err
	}

	// Step 3: create scaffolder
	scaffolder, err := options.GetScaffolder(projectConfig)
	if err != nil {
		return err
	}
	// Step 4: scaffold
	if err := scaffolder.Scaffold(); err != nil {
		return err
	}

	// Step 5: finish
	if err := options.PostScaffold(projectConfig); err != nil {
		return err
	}

	return nil
}
