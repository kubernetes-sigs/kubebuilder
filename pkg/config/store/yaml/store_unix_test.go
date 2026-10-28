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
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config/store"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// pipeDeadline is how long a store call on a named pipe may take. A call that checks the path
// returns at once, so only a call that opened the pipe is still running when the deadline passes.
const pipeDeadline = 5 * time.Second

// A named pipe blocks whoever opens it until the other end is opened too, so a store that opens
// one never returns on its own. Each call here runs with a deadline and the pipe is released
// afterwards, so a store that opened it fails the spec instead of hanging the suite.
var _ = Describe("yamlStore with a named pipe", func() {
	var s *yamlStore
	var path string

	BeforeEach(func() {
		s = New(machinery.Filesystem{FS: afero.NewOsFs()}).(*yamlStore)
		path = filepath.Join(GinkgoT().TempDir(), DefaultPath)
		Expect(syscall.Mkfifo(path, 0o600)).To(Succeed())
	})

	AfterEach(func() {
		releasePipe(path)
	})

	It("should report the pipe instead of reading it", func() {
		err := returnsWithin(pipeDeadline, func() error { return s.LoadFrom(path) })
		Expect(err).To(MatchError(occupiedConfigError(path, "not a regular file")))
		Expect(err).NotTo(MatchError(os.ErrNotExist))
	})

	It("should refuse to save a new configuration to it", func() {
		s.cfg = cfgv3.New()
		s.mustNotExist = true

		err := returnsWithin(pipeDeadline, func() error { return s.SaveTo(path) })
		Expect(err).To(MatchError(store.SaveError{
			Err: fmt.Errorf("cannot save configuration to %q: path is not a regular file", path),
		}))
	})

	It("should refuse to update a configuration at it", func() {
		s.cfg = cfgv3.New()

		err := returnsWithin(pipeDeadline, func() error { return s.SaveTo(path) })
		Expect(err).To(MatchError(store.SaveError{
			Err: fmt.Errorf("cannot save configuration to %q: path is not a regular file", path),
		}))
	})
})

// returnsWithin runs call on a goroutine of its own and returns what it returned. It fails the
// spec when call is still running once the deadline passes, which means the store opened the pipe.
func returnsWithin(deadline time.Duration, call func() error) error {
	GinkgoHelper()

	done := make(chan error, 1)
	go func() { done <- call() }()

	var err error
	Eventually(done).WithTimeout(deadline).Should(Receive(&err),
		"the store opened the named pipe and is blocked on it")

	return err
}

// releasePipe opens each end of the pipe without blocking and closes it again. A store call that
// opened the pipe is waiting for the other end, so this lets it return and its goroutine end. The
// writing end only opens while something reads the pipe, so a pipe nobody opened is left alone.
func releasePipe(path string) {
	for _, flag := range []int{os.O_RDONLY, os.O_WRONLY} {
		end, err := os.OpenFile(path, flag|syscall.O_NONBLOCK, 0)
		if err != nil {
			continue
		}
		_ = end.Close()
	}
}
