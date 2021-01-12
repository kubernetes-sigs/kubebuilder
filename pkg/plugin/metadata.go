/*
Copyright 2021 The Kubernetes Authors.

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

package plugin

// CLIMetadata is the runtime meta-data of the CLI
type CLIMetadata struct {
	// CommandName is the root command name.
	CommandName string
}

// CommandMetadata is the runtime meta-data for a command
type CommandMetadata struct {
	// Description is a description of what this command does. It is used to display help.
	Description string
	// Examples are one or more examples of the command-line usage of this command. It is used to display help.
	Examples string
}
