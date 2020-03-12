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
	"fmt"
	"testing"
)

func TestCheckGoVersion(t *testing.T) {
	invalidVersions := []string{
		"1.8", "1.8.1", "1.8.2", "1.8.3", "1.8.4", "1.8.5", "1.8.6", "1.8.7",
		"1.9", "1.9.1", "1.9.2", "1.9.3", "1.9.4", "1.9.5", "1.9.6", "1.9.7",
		"1.10", "1.10.1", "1.10.2", "1.10.3", "1.10.4", "1.10.5", "1.10.6", "1.10.7", "1.10.8",
		"1.11rc",
	}
	validVersions := []string{
		"1.11", "1.11.1", "1.11.2", "1.11.3", "1.11.4", "1.11.5", "1.11.6", "1.11.7", "1.11.8", "1.11.9",
		"1.11.10", "1.11.11", "1.11.12", "1.11.13",
		"1.12", "1.12.1", "1.12.2", "1.12.3", "1.12.4", "1.12.5", "1.12.6", "1.12.7", "1.12.8", "1.12.9",
		"1.12.10", "1.12.11", "1.12.12", "1.12.13", "1.12.14", "1.12.15", "1.12.16", "1.12.17",
		"1.13", "1.13.1", "1.13.2", "1.13.3", "1.13.4", "1.13.5", "1.13.6", "1.13.7", "1.13.8",
		"1.14",
		"1.15rc",
	}

	for _, version := range invalidVersions {
		err := checkGo(fmt.Sprintf("go version go%s linux/amd64", version), true)
		if err == nil {
			t.Errorf("version '%s' is invalid, but got no error", version)
		}
	}

	for _, version := range validVersions {
		err := checkGo(fmt.Sprintf("go version go%s linux/amd64", version), true)
		if err != nil {
			t.Errorf("version '%s' is valid, but found an error: %v", version, err)
		}
	}
}
