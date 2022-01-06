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

package store

import (
	"fmt"
)

// LoadError wraps errors yielded by Store.Load and Store.LoadFrom methods
type LoadError struct {
	Err error
}

// Error implements error interface
func (e LoadError) Error() string {
	return fmt.Sprintf("unable to load the configuration: %v", e.Err)
}

// Unwrap implements Wrapper interface
func (e LoadError) Unwrap() error {
	return e.Err
}

// SaveError wraps errors yielded by Store.Save and Store.SaveTo methods
type SaveError struct {
	Err error
}

// Error implements error interface
func (e SaveError) Error() string {
	return fmt.Sprintf("unable to save the configuration: %v", e.Err)
}

// Unwrap implements Wrapper interface
func (e SaveError) Unwrap() error {
	return e.Err
}
