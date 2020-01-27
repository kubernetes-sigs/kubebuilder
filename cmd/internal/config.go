/*
Copyright 2017 The Kubernetes Authors.

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

package internal

import (
	"log"

	"sigs.k8s.io/kubebuilder/internal/config"
)

// isProjectConfigured checks for the existence of the configuration file
func isProjectConfigured() bool {
	exists, err := config.Exists()
	if err != nil {
		log.Fatalf("Unable to check if configuration file exists: %v", err)
	}

	return exists
}

// DieIfConfigured exists if a configuration file was found
func DieIfConfigured() {
	if isProjectConfigured() {
		log.Fatalf("Project is already initialized")
	}
}

// DieIfNotConfigured exists if no configuration file was found
func DieIfNotConfigured() {
	if !isProjectConfigured() {
		log.Fatalf("Command must be run after `kubebuilder init ...`")
	}
}

// ConfiguredAndV1 returns true if the project is already configured and it is v1
func ConfiguredAndV1() bool {
	if !isProjectConfigured() {
		return false
	}

	projectConfig, err := config.Read()
	if err != nil {
		log.Fatalf("failed to read the configuration file: %v", err)
	}

	return projectConfig.IsV1()
}
