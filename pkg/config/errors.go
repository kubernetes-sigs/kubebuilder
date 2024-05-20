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

package config

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

// UnsupportedVersionError is returned by New when a project configuration version is not supported.
type UnsupportedVersionError struct {
	Version Version
}

// Error implements error interface
func (e UnsupportedVersionError) Error() string {
	return fmt.Sprintf("version %s is not supported", e.Version)
}

// UnsupportedFieldError is returned when a project configuration version does not support
// one of the fields as interface must be common for all the versions
type UnsupportedFieldError struct {
	Version Version
	Field   string
}

// Error implements error interface
func (e UnsupportedFieldError) Error() string {
	return fmt.Sprintf("version %s does not support the %s field", e.Version, e.Field)
}

// ResourceNotFoundError is returned by Config.GetResource when the provided GVK cannot be found
type ResourceNotFoundError struct {
	GVK resource.GVK
}

// Error implements error interface
func (e ResourceNotFoundError) Error() string {
	return fmt.Sprintf("resource %v could not be found", e.GVK)
}

// PluginKeyNotFoundError is returned by Config.DecodePluginConfig when the provided key cannot be found
type PluginKeyNotFoundError struct {
	Key string
}

// Error implements error interface
func (e PluginKeyNotFoundError) Error() string {
	return fmt.Sprintf("plugin key %q could not be found", e.Key)
}

// MarshalError is returned by Config.Marshal when something went wrong while marshalling to YAML
type MarshalError struct {
	Err error
}

// Error implements error interface
func (e MarshalError) Error() string {
	return fmt.Sprintf("error marshalling project configuration: %v", e.Err)
}

// Unwrap implements Wrapper interface
func (e MarshalError) Unwrap() error {
	return e.Err
}

// UnmarshalError is returned by Config.Unmarshal when something went wrong while unmarshalling from YAML
type UnmarshalError struct {
	Err error
}

// Error implements error interface
func (e UnmarshalError) Error() string {
	return fmt.Sprintf("error unmarshalling project configuration: %v", e.Err)
}

// Unwrap implements Wrapper interface
func (e UnmarshalError) Unwrap() error {
	return e.Err
}
