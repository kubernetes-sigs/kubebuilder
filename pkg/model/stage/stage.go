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

package stage

import (
	"errors"
)

var errInvalid = errors.New("invalid version stage")

// Stage represents the stability of a version
type Stage uint8

// Order Stage in decreasing degree of stability for comparison purposes.
// Stable must be 0 so that it is the default Stage
const ( // The order in this const declaration will be used to order version stages except for Stable
	// Stable should be used for plugins that are rarely changed in backwards-compatible ways, e.g. bug fixes.
	Stable Stage = iota
	// Beta should be used for plugins that may be changed in minor ways and are not expected to break between uses.
	Beta Stage = iota
	// Alpha should be used for plugins that are frequently changed and may break between uses.
	Alpha Stage = iota
)

const (
	alpha  = "alpha"
	beta   = "beta"
	stable = ""
)

// ParseStage parses stage into a Stage, assuming it is one of the valid stages
func ParseStage(stage string) (Stage, error) {
	var s Stage
	return s, s.Parse(stage)
}

// Parse parses stage inline, assuming it is one of the valid stages
func (s *Stage) Parse(stage string) error {
	switch stage {
	case alpha:
		*s = Alpha
	case beta:
		*s = Beta
	case stable:
		*s = Stable
	default:
		return errInvalid
	}
	return nil
}

// String returns the string representation of s
func (s Stage) String() string {
	switch s {
	case Alpha:
		return alpha
	case Beta:
		return beta
	case Stable:
		return stable
	default:
		panic(errInvalid)
	}
}

// Validate ensures that the stage is one of the valid stages
func (s Stage) Validate() error {
	switch s {
	case Alpha:
	case Beta:
	case Stable:
	default:
		return errInvalid
	}

	return nil
}

// Compare returns -1 if s < other, 0 if s == other, and 1 if s > other.
func (s Stage) Compare(other Stage) int {
	if s == other {
		return 0
	}

	// Stage are sorted in decreasing order
	if s > other {
		return -1
	}
	return 1
}

// IsStable returns whether the stage is stable or not
func (s Stage) IsStable() bool {
	return s == Stable
}
