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

package file

import (
	"errors"
)

// validateError is a wrapper error that will be used for errors returned by RequiresValidation.Validate
type validateError struct {
	error
}

// NewValidateError wraps an error to specify it was returned by RequiresValidation.Validate
func NewValidateError(err error) error {
	return validateError{err}
}

// Unwrap implements Wrapper interface
func (e validateError) Unwrap() error {
	return e.error
}

// IsValidateError checks if the error was returned by RequiresValidation.Validate
func IsValidateError(err error) bool {
	return errors.As(err, &validateError{})
}

// setTemplateDefaultsError is a wrapper error that will be used for errors returned by Template.SetTemplateDefaults
type setTemplateDefaultsError struct {
	error
}

// NewSetTemplateDefaultsError wraps an error to specify it was returned by Template.SetTemplateDefaults
func NewSetTemplateDefaultsError(err error) error {
	return setTemplateDefaultsError{err}
}

// Unwrap implements Wrapper interface
func (e setTemplateDefaultsError) Unwrap() error {
	return e.error
}

// IsSetTemplateDefaultsError checks if the error was returned by Template.SetTemplateDefaults
func IsSetTemplateDefaultsError(err error) bool {
	return errors.As(err, &setTemplateDefaultsError{})
}
