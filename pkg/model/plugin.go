/*
Copyright 2018 The Kubernetes Authors.

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

package model

import (
	"errors"
)

// Plugin is the interface that a plugin must implement
// We will (later) have an ExecPlugin that implements this by exec-ing a binary
type Plugin interface {
	// Pipe is the core plugin interface, that transforms a UniverseModel
	Pipe(*Universe) error
}

// pluginError is a wrapper error that will be used for errors returned by Plugin.Pipe
type pluginError struct {
	error
}

// NewPluginError wraps an error to specify it was returned by Plugin.Pipe
func NewPluginError(err error) error {
	return pluginError{err}
}

// Unwrap implements Wrapper interface
func (e pluginError) Unwrap() error {
	return e.error
}

// IsPluginError checks if the error was returned by Plugin.Pipe
func IsPluginError(err error) bool {
	return errors.As(err, &pluginError{})
}
