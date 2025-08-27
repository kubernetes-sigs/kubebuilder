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

package helpers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/h2non/gock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("helpers", func() {
	AfterEach(func() {
		gock.Off() // ensure HTTP mocks are cleared between tests
	})

	Context("BuildReleaseURL", func() {
		It("builds the exact URL for the current OS/ARCH", func() {
			v := "v4.5.0"
			expected := fmt.Sprintf(KubebuilderReleaseURL, v, runtime.GOOS, runtime.GOARCH)
			Expect(BuildReleaseURL(v)).To(Equal(expected))
		})
	})

	Context("DownloadReleaseVersionWith", func() {
		const version = "v4.6.0"

		It("downloads the binary and makes it executable", func() {
			// Arrange: mock the GitHub release endpoint
			url := BuildReleaseURL(version)
			parts := strings.SplitN(url, "/", 4)
			Expect(parts).To(HaveLen(4))
			host := parts[0] + "//" + parts[2]
			path := "/" + parts[3]

			gock.New(host).
				Get(path).
				Reply(200).
				BodyString("#!/bin/sh\necho kubebuilder\n")

			dir, err := DownloadReleaseVersionWith(version)
			Expect(err).NotTo(HaveOccurred())
			Expect(dir).NotTo(BeEmpty())

			bin := filepath.Join(dir, "kubebuilder")
			st, statErr := os.Stat(bin)
			Expect(statErr).NotTo(HaveOccurred())
			Expect(st.Mode().IsRegular()).To(BeTrue())

			if runtime.GOOS != "windows" {
				Expect(st.Mode() & 0o111).NotTo(BeZero())
			}
		})

		It("returns a clear error when the server responds non-200", func() {
			url := BuildReleaseURL(version)
			parts := strings.SplitN(url, "/", 4)
			host := parts[0] + "//" + parts[2]
			path := "/" + parts[3]

			gock.New(host).
				Get(path).
				Reply(401).
				BodyString("")

			_, err := DownloadReleaseVersionWith(version)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to download the binary: HTTP 401"))
		})

		It("propagates network errors when the request fails", func() {
			url := BuildReleaseURL(version)
			parts := strings.SplitN(url, "/", 4)
			host := parts[0] + "//" + parts[2]
			path := "/" + parts[3]

			gock.New(host).
				Get(path).
				ReplyError(errors.New("boom"))

			_, err := DownloadReleaseVersionWith(version)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to download the binary:"))
			Expect(err.Error()).To(ContainSubstring("boom"))
		})
	})
})
