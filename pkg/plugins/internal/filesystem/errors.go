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
	"errors"
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

// Error implements error interface
func (e fileExistsError) Error() string {
	return fmt.Sprintf("failed to check if %s exists: %v", e.path, e.err)
}

// Unwrap implements Wrapper interface
func (e fileExistsError) Unwrap() error {
	return e.err
}

// IsFileExistsError checks if the returned error is because the file could not be checked for existence
func IsFileExistsError(err error) bool {
	return errors.As(err, &fileExistsError{})
}

// openFileError is returned if the file could not be opened
type openFileError struct {
	path string
	err  error
}

// Error implements error interface
func (e openFileError) Error() string {
	return fmt.Sprintf("failed to open %s: %v", e.path, e.err)
}

// Unwrap implements Wrapper interface
func (e openFileError) Unwrap() error {
	return e.err
}

// IsOpenFileError checks if the returned error is because the file could not be opened
func IsOpenFileError(err error) bool {
	return errors.As(err, &openFileError{})
}

// createDirectoryError is returned if the directory could not be created
type createDirectoryError struct {
	path string
	err  error
}

// Error implements error interface
func (e createDirectoryError) Error() string {
	return fmt.Sprintf("failed to create directory for %s: %v", e.path, e.err)
}

// Unwrap implements Wrapper interface
func (e createDirectoryError) Unwrap() error {
	return e.err
}

// IsCreateDirectoryError checks if the returned error is because the directory could not be created
func IsCreateDirectoryError(err error) bool {
	return errors.As(err, &createDirectoryError{})
}

// createFileError is returned if the file could not be created
type createFileError struct {
	path string
	err  error
}

// Error implements error interface
func (e createFileError) Error() string {
	return fmt.Sprintf("failed to create %s: %v", e.path, e.err)
}

// Unwrap implements Wrapper interface
func (e createFileError) Unwrap() error {
	return e.err
}

// IsCreateFileError checks if the returned error is because the file could not be created
func IsCreateFileError(err error) bool {
	return errors.As(err, &createFileError{})
}

// readFileError is returned if the file could not be read
type readFileError struct {
	path string
	err  error
}

// Error implements error interface
func (e readFileError) Error() string {
	return fmt.Sprintf("failed to read from %s: %v", e.path, e.err)
}

// Unwrap implements Wrapper interface
func (e readFileError) Unwrap() error {
	return e.err
}

// IsReadFileError checks if the returned error is because the file could not be read
func IsReadFileError(err error) bool {
	return errors.As(err, &readFileError{})
}

// writeFileError is returned if the file could not be written
type writeFileError struct {
	path string
	err  error
}

// Error implements error interface
func (e writeFileError) Error() string {
	return fmt.Sprintf("failed to write to %s: %v", e.path, e.err)
}

// Unwrap implements Wrapper interface
func (e writeFileError) Unwrap() error {
	return e.err
}

// IsWriteFileError checks if the returned error is because the file could not be written to
func IsWriteFileError(err error) bool {
	return errors.As(err, &writeFileError{})
}

// closeFileError is returned if the file could not be created
type closeFileError struct {
	path string
	err  error
}

// Error implements error interface
func (e closeFileError) Error() string {
	return fmt.Sprintf("failed to close %s: %v", e.path, e.err)
}

// Unwrap implements Wrapper interface
func (e closeFileError) Unwrap() error {
	return e.err
}

// IsCloseFileError checks if the returned error is because the file could not be closed
func IsCloseFileError(err error) bool {
	return errors.As(err, &closeFileError{})
}
