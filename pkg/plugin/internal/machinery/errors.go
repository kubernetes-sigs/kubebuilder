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
	"errors"
	"fmt"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

// This file contains the errors returned by the scaffolding machinery
// They are not exported as they should not be created outside of this package
// Exported functions are provided to check which kind of error was returned

// fileAlreadyExistsError is returned if the file is expected not to exist but it does
type fileAlreadyExistsError struct {
	path string
}

// Error implements error interface
func (e fileAlreadyExistsError) Error() string {
	return fmt.Sprintf("failed to create %s: file already exists", e.path)
}

// IsFileAlreadyExistsError checks if the returned error is because the file already existed when expected not to
func IsFileAlreadyExistsError(err error) bool {
	return errors.As(err, &fileAlreadyExistsError{})
}

// modelAlreadyExistsError is returned if the file is expected not to exist but a previous model does
type modelAlreadyExistsError struct {
	path string
}

// Error implements error interface
func (e modelAlreadyExistsError) Error() string {
	return fmt.Sprintf("failed to create %s: model already exists", e.path)
}

// IsModelAlreadyExistsError checks if the returned error is because the model already existed when expected not to
func IsModelAlreadyExistsError(err error) bool {
	return errors.As(err, &modelAlreadyExistsError{})
}

// unknownIfExistsActionError is returned if the if-exists-action is unknown
type unknownIfExistsActionError struct {
	path           string
	ifExistsAction file.IfExistsAction
}

// Error implements error interface
func (e unknownIfExistsActionError) Error() string {
	return fmt.Sprintf("unknown behavior if file exists (%d) for %s", e.ifExistsAction, e.path)
}

// IsUnknownIfExistsActionError checks if the returned error is because the if-exists-action is unknown
func IsUnknownIfExistsActionError(err error) bool {
	return errors.As(err, &unknownIfExistsActionError{})
}
