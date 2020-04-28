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

package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/pkg/model/config"
)

const (
	// DefaultPath is the default path for the configuration file
	DefaultPath = "PROJECT"

	// DefaultVersion is the version which will be used when the version flag is not provided
	DefaultVersion = config.Version3Alpha
)

func exists(fs afero.Fs, path string) (bool, error) {
	// Look up the file
	_, err := fs.Stat(path)

	// If we could find it the file exists
	if err == nil || os.IsExist(err) {
		return true, nil
	}

	// Not existing and different errors are differentiated
	if os.IsNotExist(err) {
		err = nil
	}
	return false, err
}

func readFrom(fs afero.Fs, path string) (c config.Config, err error) {
	// Read the file
	in, err := afero.ReadFile(fs, path) //nolint:gosec
	if err != nil {
		return
	}

	// Unmarshal the file content
	if err = c.Unmarshal(in); err != nil {
		return
	}

	// kubebuilder v1 omitted version, so default to v1
	if c.Version == "" {
		c.Version = config.Version1
	}

	return
}

// Read obtains the configuration from the default path but doesn't allow to persist changes
func Read() (*config.Config, error) {
	return ReadFrom(DefaultPath)
}

// ReadFrom obtains the configuration from the provided path but doesn't allow to persist changes
func ReadFrom(path string) (*config.Config, error) {
	c, err := readFrom(afero.NewOsFs(), path)
	return &c, err
}

// Config extends model/config.Config allowing to persist changes
// NOTE: the existence of Config structs in both model and internal packages is to guarantee that kubebuilder
// is the only project that can modify the file, while plugins can still receive the configuration
type Config struct {
	config.Config

	// path stores where the config should be saved to
	path string
	// mustNotExist requires the file not to exist when saving it
	mustNotExist bool
	// fs is for testing.
	fs afero.Fs
}

// New creates a new configuration that will be stored at the provided path
func New(path string) *Config {
	return &Config{
		Config: config.Config{
			Version: DefaultVersion,
		},
		path:         path,
		mustNotExist: true,
		fs:           afero.NewOsFs(),
	}
}

// Load obtains the configuration from the default path allowing to persist changes (Save method)
func Load() (*Config, error) {
	return LoadFrom(DefaultPath)
}

// LoadInitialized calls Load() but returns helpful error messages if the config
// does not exist.
func LoadInitialized() (*Config, error) {
	c, err := Load()
	if os.IsNotExist(err) {
		return nil, errors.New("unable to find configuration file, project must be initialized")
	}
	return c, err
}

// LoadFrom obtains the configuration from the provided path allowing to persist changes (Save method)
func LoadFrom(path string) (*Config, error) {
	fs := afero.NewOsFs()
	c, err := readFrom(fs, path)
	return &Config{Config: c, path: path, fs: fs}, err
}

// Save saves the configuration information
func (c Config) Save() error {
	if c.fs == nil {
		c.fs = afero.NewOsFs()
	}
	// If path is unset, it was created directly with `Config{}`
	if c.path == "" {
		return saveError{errors.New("no information where it should be stored, " +
			"use one of the constructors (`New`, `Load` or `LoadFrom`) to create Config instances")}
	}

	// If it is a new configuration, the path should not exist yet
	if c.mustNotExist {
		// Lets check that the file doesn't exist
		alreadyExists, err := exists(c.fs, c.path)
		if err != nil {
			return saveError{err}
		}
		if alreadyExists {
			return saveError{errors.New("configuration already exists in the provided path")}
		}
	}

	// Marshall into YAML
	content, err := c.Marshal()
	if err != nil {
		return saveError{err}
	}

	// Write the marshalled configuration
	err = afero.WriteFile(c.fs, c.path, content, 0600)
	if err != nil {
		return saveError{fmt.Errorf("failed to save configuration to %s: %v", c.path, err)}
	}

	return nil
}

// Path returns the path for configuration file
func (c Config) Path() string {
	return c.path
}

type saveError struct {
	err error
}

func (e saveError) Error() string {
	return fmt.Sprintf("unable to save the configuration: %v", e.err)
}
