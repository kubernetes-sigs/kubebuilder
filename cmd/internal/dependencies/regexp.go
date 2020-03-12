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

package dependencies

import (
	"regexp"
)

// FindStringSubmatch works the same as regexp.Regexp.FindStringSubmatch
//but parses the output to a map binding names with matches
func FindStringSubmatch(re *regexp.Regexp, input string) (output map[string]string) {
	match := re.FindStringSubmatch(input)
	if match == nil {
		return
	}

	output = make(map[string]string)
	for i, name := range re.SubexpNames() {
		if name != "" {
			output[name] = match[i]
		}
	}
	return
}
