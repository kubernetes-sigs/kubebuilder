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

package scaffold

import (
	"fmt"
	"strings"
)

// Pattern is the enumerated type for patterns
type Pattern string

const (
	// PatternNone is the pattern constant for standard generation
	PatternNone = ""

	// PatternAddon is the pattern constant for addons
	PatternAddon = "addon"
)

// PatternAllValues is the list of valid pattern values
var PatternAllValues = []Pattern{PatternNone, PatternAddon}

// Get is part of the implementation of flag.Getter
func (p *Pattern) Get() interface{} {
	return *p
}

// String is part of the implementation of flag.Value
func (p *Pattern) String() string {
	return string(*p)
}

// Set is part of the implentation of flag.Value
func (p *Pattern) Set(s string) error {
	*p = Pattern(s)
	return nil
}

// Type is part of the implentation of pflag.Value
func (p *Pattern) Type() string {
	return "pattern"
}

// Validate checks the Pattern value to make sure it is valid.
func (p *Pattern) Validate() error {
	if p == nil {
		return nil
	}

	{
		found := false
		for _, f := range PatternAllValues {
			if f == *p {
				found = true
				continue
			}
		}
		if !found {
			var patternStrings []string
			for _, f := range PatternAllValues {
				patternStrings = append(patternStrings, string(f))
			}
			return fmt.Errorf("Pattern %q is not recognized, must be one of %s", *p, strings.Join(patternStrings, ","))
		}
	}

	return nil
}
