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

package store

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfigStore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Store Suite")
}

var _ = Describe("LoadError", func() {
	var (
		wrapped = fmt.Errorf("error message")
		err     = LoadError{Err: wrapped}
	)

	Context("Error", func() {
		It("should return the correct error message", func() {
			Expect(err.Error()).To(Equal(fmt.Sprintf("unable to load the configuration: %v", wrapped)))
		})
	})

	Context("Unwrap", func() {
		It("should unwrap to the wrapped error", func() {
			Expect(err.Unwrap()).To(Equal(wrapped))
		})
	})
})

var _ = Describe("SaveError", func() {
	var (
		wrapped = fmt.Errorf("error message")
		err     = SaveError{Err: wrapped}
	)

	Context("Error", func() {
		It("should return the correct error message", func() {
			Expect(err.Error()).To(Equal(fmt.Sprintf("unable to save the configuration: %v", wrapped)))
		})
	})

	Context("Unwrap", func() {
		It("should unwrap to the wrapped error", func() {
			Expect(err.Unwrap()).To(Equal(wrapped))
		})
	})
})
