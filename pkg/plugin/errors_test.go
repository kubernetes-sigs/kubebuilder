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

package plugin

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PluginKeyNotFoundError", func() {
	err := ExitError{
		Plugin: "go.kubebuilder.io/v1",
		Reason: "skipping plugin",
	}

	Context("Error", func() {
		It("should return the correct error message", func() {
			Expect(err.Error()).To(Equal("plugin \"go.kubebuilder.io/v1\" exit early: skipping plugin"))
		})
	})
})
