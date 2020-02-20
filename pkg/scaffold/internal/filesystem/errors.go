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

package filesystem

import (
	"fmt"
)

// This file contains the errors returned by the file system wrapper
// They are not exported as they should not be created outside of this package
// Exported functions are provided to check which kind of error was returned

// fileExistsError is returned if it could not be checked if the file exists
type fileExistsError struct {
	path string
	err  error
}

func (e fileExistsError) Error() string {
	return fmt.Sprintf("failed to check if %s exists: %v", e.path, e.err)
}

// IsFileExistsError checks if the returned error is because the file could not be checked for existence
func IsFileExistsError(e error) bool {
	_, ok := e.(fileExistsError)
	return ok
}

// openFileError is returned if the file could not be opened
type openFileError struct {
	path string
	err  error
}

func (e openFileError) Error() string {
	return fmt.Sprintf("failed to open %s: %v", e.path, e.err)
}

// IsOpenFileError checks if the returned error is because the file could not be opened
func IsOpenFileError(e error) bool {
	_, ok := e.(openFileError)
	return ok
}

// createDirectoryError is returned if the directory could not be created
type createDirectoryError struct {
	path string
	err  error
}

func (e createDirectoryError) Error() string {
	return fmt.Sprintf("failed to create directory for %s: %v", e.path, e.err)
}

// IsCreateDirectoryError checks if the returned error is because the directory could not be created
func IsCreateDirectoryError(e error) bool {
	_, ok := e.(createDirectoryError)
	return ok
}

// createFileError is returned if the file could not be created
type createFileError struct {
	path string
	err  error
}

func (e createFileError) Error() string {
	return fmt.Sprintf("failed to create %s: %v", e.path, e.err)
}

// IsCreateFileError checks if the returned error is because the file could not be created
func IsCreateFileError(e error) bool {
	_, ok := e.(createFileError)
	return ok
}

// readFileError is returned if the file could not be read
type readFileError struct {
	path string
	err  error
}

func (e readFileError) Error() string {
	return fmt.Sprintf("failed to read from %s: %v", e.path, e.err)
}

// IsReadFileError checks if the returned error is because the file could not be read
func IsReadFileError(e error) bool {
	_, ok := e.(readFileError)
	return ok
}

// writeFileError is returned if the file could not be written
type writeFileError struct {
	path string
	err  error
}

func (e writeFileError) Error() string {
	return fmt.Sprintf("failed to write to %s: %v", e.path, e.err)
}

// IsWriteFileError checks if the returned error is because the file could not be written
func IsWriteFileError(e error) bool {
	_, ok := e.(writeFileError)
	return ok
}

// closeFileError is returned if the file could not be created
type closeFileError struct {
	path string
	err  error
}

func (e closeFileError) Error() string {
	return fmt.Sprintf("failed to close %s: %v", e.path, e.err)
}

// IsCloseFileError checks if the returned error is because the file could not be closed
func IsCloseFileError(e error) bool {
	_, ok := e.(closeFileError)
	return ok
}
