/*
Copyright 2025 The Kubernetes Authors.

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

//lint:ignore ST1001 we use dot-imports in tests for brevity

package alpha

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Figuring out ways to test run these tests similar to existing.
// Currently unable to run without this on VSCode. Will remove once done
func TestCommand(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "suite test for alpha commands")
}
