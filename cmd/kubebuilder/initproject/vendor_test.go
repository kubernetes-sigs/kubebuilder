/*
Copyright 2017 The Kubernetes Authors.

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

package initproject

import (
	"bytes"
	"testing"
)

func TestTryReadingUserDeps(t *testing.T) {
	tests := []struct {
		in          string
		expKBMarker bool
		expUserDeps string
	}{
		{
			in: `
ABC
ABC
jslsls
sjsslsls
			`,
			expKBMarker: false,
			expUserDeps: "",
		},
		{
			in: `
ABC
ABC
# DO NOT MODIFY BELOW THIS LINE.
jslsls
sjsslsls
			`,
			expKBMarker: true,
			expUserDeps: `
ABC
ABC
# DO NOT MODIFY BELOW THIS LINE.
`,
		},
		{
			in: `
ABC
ABC
# DO NOT MODIFY BELOW THIS LINE.
			`,
			expKBMarker: true,
			expUserDeps: `
ABC
ABC
# DO NOT MODIFY BELOW THIS LINE.
`,
		},
	}

	for _, test := range tests {
		r := bytes.NewReader([]byte(test.in))
		userDeps, kbMarker, err := tryReadingUserDeps(r)
		if err != nil {
			t.Errorf("Reading UserDeps should succeed, but got an error: %v", err)
		}
		if test.expKBMarker != kbMarker {
			t.Errorf("KB marker mismatch: exp: '%v' got: '%v'", test.expKBMarker, kbMarker)
		}
		if test.expUserDeps != userDeps {
			t.Errorf("UserDeps don't match: exp: '%v' got: '%v'", test.expUserDeps, userDeps)
		}

	}
}
