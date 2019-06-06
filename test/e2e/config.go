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

package e2e

import (
	"path/filepath"
)

// runtime config specified to run e2e tests
type config struct {
	domain              string
	group               string
	version             string
	kind                string
	controllerImageName string
	workDir             string
}

// configWithSuffix init with a random suffix for test config stuff,
// to avoid conflict when running tests synchronously.
func configWithSuffix(testSuffix string) (*config, error) {
	testGroup := "bar" + testSuffix
	path, err := filepath.Abs("e2e-" + testSuffix)
	if err != nil {
		return nil, err
	}

	return &config{
		domain:              "example.com" + testSuffix,
		group:               testGroup,
		version:             "v1alpha1",
		kind:                "Foo" + testSuffix,
		controllerImageName: "e2e-test/controller-manager:" + testSuffix,
		workDir:             path,
	}, nil
}
