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

package machinery

import (
	"errors"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func TestErrors(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Error suite")
}

var _ = Describe("Errors", func() {
	var (
		path                     = filepath.Join("path", "to", "file")
		err                      = errors.New("test error")
		fileAlreadyExistsErr     = fileAlreadyExistsError{path}
		modelAlreadyExistsErr    = modelAlreadyExistsError{path}
		unknownIfExistsActionErr = unknownIfExistsActionError{path, -1}
	)

	DescribeTable("IsXxxxError should return true for themselves and false for the rest",
		func(f func(error) bool, itself error, rest ...error) {
			Expect(f(itself)).To(BeTrue())
			for _, err := range rest {
				Expect(f(err)).To(BeFalse())
			}
		},
		Entry("file exists", IsFileAlreadyExistsError, fileAlreadyExistsErr,
			err, modelAlreadyExistsErr, unknownIfExistsActionErr),
		Entry("model exists", IsModelAlreadyExistsError, modelAlreadyExistsErr,
			err, fileAlreadyExistsErr, unknownIfExistsActionErr),
		Entry("unknown if exists action", IsUnknownIfExistsActionError, unknownIfExistsActionErr,
			err, fileAlreadyExistsErr, modelAlreadyExistsErr),
	)

	DescribeTable("should contain the wrapped error and error message",
		func(err error) {
			Expect(err).To(MatchError(err))
			Expect(err.Error()).To(ContainSubstring(err.Error()))
		},
	)

	// NOTE: the following test increases coverage
	It("should print a descriptive error message", func() {
		Expect(fileAlreadyExistsErr.Error()).To(ContainSubstring("file already exists"))
		Expect(modelAlreadyExistsErr.Error()).To(ContainSubstring("model already exists"))
		Expect(unknownIfExistsActionErr.Error()).To(ContainSubstring("unknown behavior if file exists"))
	})
})
