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

package store

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
)

// Store represents a persistence backend for config.Config
type Store interface {
	// New creates a new config.Config to store
	New(config.Version) error
	// Load retrieves the config.Config from the persistence backend
	Load() error
	// LoadFrom retrieves the config.Config from the persistence backend at the specified key
	LoadFrom(string) error
	// Save stores the config.Config into the persistence backend
	Save() error
	// SaveTo stores the config.Config into the persistence backend at the specified key
	SaveTo(string) error

	// Config returns the stored config.Config
	Config() config.Config
}
