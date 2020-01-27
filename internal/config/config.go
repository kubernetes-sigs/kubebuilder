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
	"io/ioutil"
	"os"

	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/yaml"
)

// Default path for the configuration file
const DefaultPath = "PROJECT"

func exists(path string) (bool, error) {
	// Look up the file
	_, err := os.Stat(path)

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

// Exists verifies that the configuration file exists in the default path
// TODO: consider removing this verification in favor of using Load and checking the error
func Exists() (bool, error) {
	return exists(DefaultPath)
}

func readFrom(path string) (c config.Config, err error) {
	// Read the file
	in, err := ioutil.ReadFile(path) // nolint: gosec
	if err != nil {
		return
	}

	// Unmarshal the file content
	if err = yaml.Unmarshal(in, &c); err != nil {
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
	c, err := readFrom(path)

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
}

// New creates a new configuration that will be stored at the provided path
// TODO: this method should be used during the initialization command, unused for now
func New(path string) *Config {
	return &Config{
		Config: config.Config{
			Version: config.Version2,
		},
		path:         path,
		mustNotExist: true,
	}
}

// Load obtains the configuration from the default path allowing to persist changes (Save method)
func Load() (*Config, error) {
	return LoadFrom(DefaultPath)
}

// LoadFrom obtains the configuration from the provided path allowing to persist changes (Save method)
func LoadFrom(path string) (*Config, error) {
	c, err := readFrom(path)

	return &Config{Config: c, path: path}, err
}

// Save saves the configuration information
func (c Config) Save() error {
	// If path is unset, it was created directly with `Config{}`
	if c.path == "" {
		return saveError{errors.New("no information where it should be stored, " +
			"use one of the constructors (`New`, `Load` or `LoadFrom`) to create Config instances")}
	}

	// If it is a new configuration, the path should not exist yet
	if c.mustNotExist {
		// Lets check that the file doesn't exist
		alreadyExists, err := exists(c.path)
		if err != nil {
			return saveError{err}
		}
		if alreadyExists {
			return saveError{errors.New("configuration already exists in the provided path")}
		}
	}

	// Marshall into YAML
	content, err := yaml.Marshal(c)
	if err != nil {
		return saveError{fmt.Errorf("error marshalling project configuration: %v", err)}
	}

	// Write the marshalled configuration
	err = ioutil.WriteFile(c.path, content, os.ModePerm)
	if err != nil {
		return saveError{fmt.Errorf("failed to save configuration to %s: %v", c.path, err)}
	}

	return nil
}

type saveError struct {
	err error
}

func (e saveError) Error() string {
	return fmt.Sprintf("unable to save the configuration: %v", e.err)
}
