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

package machinery

import (
	"fmt"
)

// This file contains the errors returned by the scaffolding machinery
// They are exported to be able to check which kind of error was returned

// ValidateError is a wrapper error that will be used for errors returned by RequiresValidation.Validate
type ValidateError struct {
	error
}

// Unwrap implements Wrapper interface
func (e ValidateError) Unwrap() error {
	return e.error
}

// SetTemplateDefaultsError is a wrapper error that will be used for errors returned by Template.SetTemplateDefaults
type SetTemplateDefaultsError struct {
	error
}

// Unwrap implements Wrapper interface
func (e SetTemplateDefaultsError) Unwrap() error {
	return e.error
}

// ExistsFileError is a wrapper error that will be used for errors when checking for a file existence
type ExistsFileError struct {
	error
}

// Unwrap implements Wrapper interface
func (e ExistsFileError) Unwrap() error {
	return e.error
}

// OpenFileError is a wrapper error that will be used for errors when opening a file
type OpenFileError struct {
	error
}

// Unwrap implements Wrapper interface
func (e OpenFileError) Unwrap() error {
	return e.error
}

// CreateDirectoryError is a wrapper error that will be used for errors when creating a directory
type CreateDirectoryError struct {
	error
}

// Unwrap implements Wrapper interface
func (e CreateDirectoryError) Unwrap() error {
	return e.error
}

// CreateFileError is a wrapper error that will be used for errors when creating a file
type CreateFileError struct {
	error
}

// Unwrap implements Wrapper interface
func (e CreateFileError) Unwrap() error {
	return e.error
}

// ReadFileError is a wrapper error that will be used for errors when reading a file
type ReadFileError struct {
	error
}

// Unwrap implements Wrapper interface
func (e ReadFileError) Unwrap() error {
	return e.error
}

// WriteFileError is a wrapper error that will be used for errors when writing a file
type WriteFileError struct {
	error
}

// Unwrap implements Wrapper interface
func (e WriteFileError) Unwrap() error {
	return e.error
}

// CloseFileError is a wrapper error that will be used for errors when closing a file
type CloseFileError struct {
	error
}

// Unwrap implements Wrapper interface
func (e CloseFileError) Unwrap() error {
	return e.error
}

// ModelAlreadyExistsError is returned if the file is expected not to exist but a previous model does
type ModelAlreadyExistsError struct {
	path string
}

// Error implements error interface
func (e ModelAlreadyExistsError) Error() string {
	return fmt.Sprintf("failed to create %s: model already exists", e.path)
}

// UnknownIfExistsActionError is returned if the if-exists-action is unknown
type UnknownIfExistsActionError struct {
	path           string
	ifExistsAction IfExistsAction
}

// Error implements error interface
func (e UnknownIfExistsActionError) Error() string {
	return fmt.Sprintf("unknown behavior if file exists (%d) for %s", e.ifExistsAction, e.path)
}

// FileAlreadyExistsError is returned if the file is expected not to exist but it does
type FileAlreadyExistsError struct {
	path string
}

// Error implements error interface
func (e FileAlreadyExistsError) Error() string {
	return fmt.Sprintf("failed to create %s: file already exists", e.path)
}
