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

package yaml

import (
	"fmt"
	"os"

	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/config/store"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

const (
	// DefaultPath is the default path for the configuration file
	DefaultPath = "PROJECT"
)

// yamlStore implements store.Store using a YAML file as the storage backend
// The key is translated into the YAML file path
type yamlStore struct {
	// fs is the filesystem that will be used to store the config.Config
	fs afero.Fs
	// mustNotExist requires the file not to exist when saving it
	mustNotExist bool

	cfg config.Config
}

// New creates a new configuration that will be stored at the provided path
func New(fs machinery.Filesystem) store.Store {
	return &yamlStore{fs: fs.FS}
}

// New implements store.Store interface
func (s *yamlStore) New(version config.Version) error {
	cfg, err := config.New(version)
	if err != nil {
		return err
	}

	s.cfg = cfg
	s.mustNotExist = true
	return nil
}

// Load implements store.Store interface
func (s *yamlStore) Load() error {
	return s.LoadFrom(DefaultPath)
}

type versionedConfig struct {
	Version config.Version `json:"version"`
}

// LoadFrom implements store.Store interface
func (s *yamlStore) LoadFrom(path string) error {
	s.mustNotExist = false

	// Read the file
	in, err := afero.ReadFile(s.fs, path)
	if err != nil {
		return store.LoadError{Err: fmt.Errorf("unable to read %q file: %w", path, err)}
	}

	// Check the file version
	var versioned versionedConfig
	if err := yaml.Unmarshal(in, &versioned); err != nil {
		return store.LoadError{Err: fmt.Errorf("unable to determine config version: %w", err)}
	}

	// Create the config object
	var cfg config.Config
	cfg, err = config.New(versioned.Version)
	if err != nil {
		return store.LoadError{Err: fmt.Errorf("unable to create config for version %q: %w", versioned.Version, err)}
	}

	// Unmarshal the file content
	if err := cfg.UnmarshalYAML(in); err != nil {
		return store.LoadError{Err: fmt.Errorf("unable to unmarshal config at %q: %w", path, err)}
	}

	s.cfg = cfg
	return nil
}

// Save implements store.Store interface
func (s yamlStore) Save() error {
	return s.SaveTo(DefaultPath)
}

// SaveTo implements store.Store interface
func (s yamlStore) SaveTo(path string) error {
	// If yamlStore is unset, none of New, Load, or LoadFrom were called successfully
	if s.cfg == nil {
		return store.SaveError{Err: fmt.Errorf("undefined config, use one of the initializers: New, Load, LoadFrom")}
	}

	// If it is a new configuration, the path should not exist yet
	if s.mustNotExist {
		// Lets check that the file doesn't exist
		_, err := s.fs.Stat(path)
		if os.IsNotExist(err) {
			// This is exactly what we want
		} else if err == nil || os.IsExist(err) {
			return store.SaveError{Err: fmt.Errorf("configuration already exists in %q", path)}
		} else {
			return store.SaveError{Err: fmt.Errorf("unable to check for file prior existence: %w", err)}
		}
	}

	// Marshall into YAML
	content, err := s.cfg.MarshalYAML()
	if err != nil {
		return store.SaveError{Err: fmt.Errorf("unable to marshal to YAML: %w", err)}
	}

	// Write the marshalled configuration
	err = afero.WriteFile(s.fs, path, content, 0600)
	if err != nil {
		return store.SaveError{Err: fmt.Errorf("failed to save configuration to %q: %w", path, err)}
	}

	return nil
}

// Config implements store.Store interface
func (s yamlStore) Config() config.Config {
	return s.cfg
}
