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
	"fmt"
	"os"

	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
)

const (
	// DefaultPath is the default path for the configuration file
	DefaultPath = "PROJECT"
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

type versionedConfig struct {
	Version config.Version `json:"version"`
}

func readFrom(fs afero.Fs, path string) (config.Config, error) {
	// Read the file
	in, err := afero.ReadFile(fs, path)
	if err != nil {
		return nil, err
	}

	// Check the file version
	var versioned versionedConfig
	if err := yaml.Unmarshal(in, &versioned); err != nil {
		return nil, err
	}

	// Create the config object
	var c config.Config
	c, err = config.New(versioned.Version)
	if err != nil {
		return nil, err
	}

	// Unmarshal the file content
	if err := c.Unmarshal(in); err != nil {
		return nil, err
	}

	return c, nil
}

// Read obtains the configuration from the default path but doesn't allow to persist changes
func Read(fs afero.Fs) (config.Config, error) {
	return ReadFrom(fs, DefaultPath)
}

// ReadFrom obtains the configuration from the provided path but doesn't allow to persist changes
func ReadFrom(fs afero.Fs, path string) (config.Config, error) {
	return readFrom(fs, path)
}

// Config extends config.Config allowing to persist changes
// NOTE: the existence of Config structs in both model and internal packages is to guarantee that kubebuilder
// is the only project that can modify the file, while plugins can still receive the configuration
type Config struct {
	// fs is the filesystem that the Config backend will use to store the config.Config
	fs afero.Fs
	// path stores where the config should be saved to
	path string
	// mustNotExist requires the file not to exist when saving it
	mustNotExist bool

	config.Config
}

// New creates a new configuration that will be stored at the provided path
func New(fs afero.Fs) *Config {
	return &Config{
		path: DefaultPath,
		fs:   fs,
	}
}

// Init initializes a new Config for the provided config.Version
func (c *Config) Init(version config.Version) error {
	cfg, err := config.New(version)
	if err != nil {
		return err
	}

	c.Config = cfg
	c.mustNotExist = true
	return nil
}

// InitTo initializes a new Config for the provided config.Version to the provided path
func (c *Config) InitTo(path string, version config.Version) error {
	c.path = path
	return c.Init(version)
}

// Load obtains the configuration from the default path allowing to persist changes (Save method)
func (c *Config) Load() error {
	c.mustNotExist = false

	cfg, err := readFrom(c.fs, c.path)
	if err != nil {
		return err
	}

	c.Config = cfg
	return nil
}

// LoadFrom obtains the configuration from the provided path allowing to persist changes (Save method)
func (c *Config) LoadFrom(path string) error {
	c.path = path
	return c.Load()
}

// Save saves the configuration information
func (c Config) Save() error {
	// TODO: instead of exposing Config and checking that the expected use with New and one of Init, InitTo, Load,
	//       or LoadFrom was used (by checking that the unexported fields fs and Config are set), it would be nicer to
	//       expose an interface and a constructor and not expose the type.
	// If fs is unset, it was created directly with `Config{}`
	if c.fs == nil {
		return saveError{fmt.Errorf("undefined filesystem, use the constructor New to create Config instances")}
	}
	// If Config is unset, none of Init, InitTo, Load, or LoadFrom were called successfully
	if c.Config == nil {
		return saveError{fmt.Errorf("undefined Config, use one of the initializers: Init, InitTo, Load, LoadFrom")}
	}

	// If it is a new configuration, the path should not exist yet
	if c.mustNotExist {
		// Lets check that the file doesn't exist
		alreadyExists, err := exists(c.fs, c.path)
		if err != nil {
			return saveError{fmt.Errorf("unable to check for file prior existence: %w", err)}
		}
		if alreadyExists {
			return saveError{fmt.Errorf("configuration already exists in the provided path")}
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
		return saveError{fmt.Errorf("failed to save configuration to %s: %w", c.path, err)}
	}

	return nil
}

type saveError struct {
	err error
}

func (e saveError) Error() string {
	return fmt.Sprintf("unable to save the configuration: %v", e.err)
}

func (e saveError) Unwrap() error {
	return e.err
}
