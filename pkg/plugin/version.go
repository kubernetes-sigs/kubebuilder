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

package plugin

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
)

var (
	errNegative = errors.New("plugin version number must be positive")
	errEmpty    = errors.New("plugin version is empty")
)

// Version is a plugin version containing a positive integer and a stage value that represents stability.
type Version struct {
	// Number denotes the current version of a plugin. Two different numbers between versions
	// indicate that they are incompatible.
	Number int
	// Stage indicates stability.
	Stage stage.Stage
}

// Parse parses version inline, assuming it adheres to format: (v)?[0-9]*(-(alpha|beta))?
func (v *Version) Parse(version string) error {
	version = strings.TrimPrefix(version, "v")
	if len(version) == 0 {
		return errEmpty
	}

	substrings := strings.SplitN(version, "-", 2)

	var err error
	if v.Number, err = strconv.Atoi(substrings[0]); err != nil {
		// Lets check if the `-` belonged to a negative number
		if n, err := strconv.Atoi(version); err == nil && n < 0 {
			return errNegative
		}
		return err
	}

	if len(substrings) > 1 {
		if err = v.Stage.Parse(substrings[1]); err != nil {
			return err
		}
	}

	return nil
}

// String returns the string representation of v.
func (v Version) String() string {
	stageStr := v.Stage.String()
	if len(stageStr) == 0 {
		return fmt.Sprintf("v%d", v.Number)
	}
	return fmt.Sprintf("v%d-%s", v.Number, stageStr)
}

// Validate ensures that the version number is positive and the stage is one of the valid stages.
func (v Version) Validate() error {
	if v.Number < 0 {
		return errNegative
	}

	return v.Stage.Validate()
}

// Compare returns -1 if v < other, 0 if v == other, and 1 if v > other.
func (v Version) Compare(other Version) int {
	if v.Number > other.Number {
		return 1
	} else if v.Number < other.Number {
		return -1
	}

	return v.Stage.Compare(other.Stage)
}

// IsStable returns true if v is stable.
func (v Version) IsStable() bool {
	// Plugin version 0 is not considered stable
	if v.Number == 0 {
		return false
	}

	// Any other version than 0 depends on its stage field
	return v.Stage.IsStable()
}
