//go:build !windows

/*
Copyright 2026 The Kubernetes Authors.

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

package yaml

import (
	"os"
	"path/filepath"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// A named pipe blocks the reader until something writes to it, so a store that opens one never
// returns. These specs hang instead of failing when that happens.
var _ = Describe("yamlStore with a named pipe", func() {
	var s *yamlStore
	var path string

	BeforeEach(func() {
		s = New(machinery.Filesystem{FS: afero.NewOsFs()}).(*yamlStore)
		path = filepath.Join(GinkgoT().TempDir(), DefaultPath)
		Expect(syscall.Mkfifo(path, 0o600)).To(Succeed())
	})

	It("should report the pipe instead of reading it", func() {
		err := s.LoadFrom(path)
		Expect(err).To(MatchError(occupiedConfigError(path, "not a regular file")))
		Expect(err).NotTo(MatchError(os.ErrNotExist))
	})

	It("should refuse to save a new configuration to it", func() {
		s.cfg = cfgv3.New()
		s.mustNotExist = true

		Expect(s.SaveTo(path)).NotTo(Succeed())
	})

	It("should refuse to update a configuration at it", func() {
		s.cfg = cfgv3.New()

		Expect(s.SaveTo(path)).NotTo(Succeed())
	})
})
