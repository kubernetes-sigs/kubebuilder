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

package config

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var _ = Describe("UnsupportedVersionError", func() {
	err := UnsupportedVersionError{
		Version: Version{Number: 1},
	}

	Context("Error", func() {
		It("should return the correct error message", func() {
			Expect(err.Error()).To(Equal("version 1 is not supported"))
		})
	})
})

var _ = Describe("UnsupportedFieldError", func() {
	err := UnsupportedFieldError{
		Version: Version{Number: 1},
		Field:   "name",
	}

	Context("Error", func() {
		It("should return the correct error message", func() {
			Expect(err.Error()).To(Equal("version 1 does not support the name field"))
		})
	})
})

var _ = Describe("ResourceNotFoundError", func() {
	err := ResourceNotFoundError{
		GVK: resource.GVK{
			Group:   "group",
			Domain:  "my.domain",
			Version: "v1",
			Kind:    "Kind",
		},
	}

	Context("Error", func() {
		It("should return the correct error message", func() {
			Expect(err.Error()).To(Equal("resource {group my.domain v1 Kind} could not be found"))
		})
	})
})

var _ = Describe("PluginKeyNotFoundError", func() {
	err := PluginKeyNotFoundError{
		Key: "go.kubebuilder.io/v1",
	}

	Context("Error", func() {
		It("should return the correct error message", func() {
			Expect(err.Error()).To(Equal("plugin key \"go.kubebuilder.io/v1\" could not be found"))
		})
	})
})

var _ = Describe("MarshalError", func() {
	var (
		wrapped = fmt.Errorf("error message")
		err     = MarshalError{Err: wrapped}
	)

	Context("Error", func() {
		It("should return the correct error message", func() {
			Expect(err.Error()).To(Equal(fmt.Sprintf("error marshalling project configuration: %v", wrapped)))
		})
	})

	Context("Unwrap", func() {
		It("should unwrap to the wrapped error", func() {
			Expect(err.Unwrap()).To(Equal(wrapped))
		})
	})
})

var _ = Describe("UnmarshalError", func() {
	var (
		wrapped = fmt.Errorf("error message")
		err     = UnmarshalError{Err: wrapped}
	)

	Context("Error", func() {
		It("should return the correct error message", func() {
			Expect(err.Error()).To(Equal(fmt.Sprintf("error unmarshalling project configuration: %v", wrapped)))
		})
	})

	Context("Unwrap", func() {
		It("should unwrap to the wrapped error", func() {
			Expect(err.Unwrap()).To(Equal(wrapped))
		})
	})
})
