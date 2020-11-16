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

package util

import (
	"testing"
)

func TestCheckGoVersion(t *testing.T) {

	tests := []struct {
		ver       string
		isInvalid bool
	}{
		{"go1.8", true},
		{"go1.9", true},
		{"go1.10", true},
		{"go1.11", true},
		{"go1.11rc", true},
		{"go1.11.1", true},
		{"go1.12rc2", true},
		{"go1.13", false},
	}

	for _, test := range tests {
		err := checkGoVersion(test.ver)
		if err != nil {
			// go error, but the version isn't invalid
			if !test.isInvalid {
				t.Errorf("Go version check failed valid version '%s' with error '%s'", test.ver, err)
			}
		} else {
			// got no error, but the version is invalid
			if test.isInvalid {
				t.Errorf("version '%s' is invalid, but got no error", test.ver)
			}
		}
	}

}
