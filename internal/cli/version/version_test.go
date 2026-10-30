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

package version

import (
	"runtime/debug"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	testGitCommit             = "abcdef123"
	testGitCommitDirty        = "abcdef123-dirty"
	testKubebuilderV410       = "v4.10.4"
	testKubebuilderVersion135 = "1.35.0"
)

var _ = Describe("Version", func() {
	Context("New", func() {
		It("should use the version from the environment variable override", func() {
			expectedVersion := "v9.9.9-test"
			GinkgoT().Setenv("KUBEBUILDER_TEST_VERSION", expectedVersion)

			v := New()

			Expect(v.KubeBuilderVersion).To(Equal(expectedVersion))
			Expect(v.GitCommit).To(Equal("test-commit"))
		})

		It("should fill the Kubernetes vendor version and the runtime platform", func() {
			v := New()

			Expect(v.KubernetesVendor).To(Equal(kubernetesVendorVersion))
			Expect(v.GoOs).NotTo(BeEmpty())
			Expect(v.GoArch).NotTo(BeEmpty())
		})

		It("should keep a tagged release version when the build is dirty", func() {
			v := &Version{KubeBuilderVersion: "v1.0.0"}
			settings := []debug.BuildSetting{
				{Key: vcsKeyRevision, Value: testGitCommit},
				{Key: vcsKeyModified, Value: vcsValueTrue},
			}

			v.applyVCSMetadata(settings)

			Expect(v.GitCommit).To(HaveSuffix("-dirty"))
			// For tagged releases, we ignore dirty flag to support GoReleaser builds
			Expect(v.KubeBuilderVersion).To(Equal("v1.0.0"))
		})

		It("should format the version string without the v prefix", func() {
			v := Version{KubeBuilderVersion: "v1.2.3"}

			Expect(v.GetKubeBuilderVersion()).To(Equal("1.2.3"))
		})
	})

	DescribeTable("GetKubeBuilderVersion",
		func(input, expected string) {
			v := Version{KubeBuilderVersion: input}
			Expect(v.GetKubeBuilderVersion()).To(Equal(expected))
		},
		Entry("strips v prefix", "v"+testKubebuilderVersion135, testKubebuilderVersion135),
		Entry("handles no prefix", testKubebuilderVersion135, testKubebuilderVersion135),
		Entry("preserves devel", develVersion, develVersion),
		Entry("handles empty", "", ""),
		Entry("handles dirty suffix", "v"+testKubebuilderVersion135+"-dirty", testKubebuilderVersion135+"-dirty"),
	)

	DescribeTable("resolveMainVersion",
		func(input, expected string) {
			Expect(resolveMainVersion(debug.Module{Version: input})).To(Equal(expected))
		},
		Entry("Valid Tag", testKubebuilderV410, testKubebuilderV410),
		Entry("Development Build", "", develVersion),
	)

	DescribeTable("applyVCSMetadata",
		func(initialVersion string, settings []debug.BuildSetting, expectCommit, expectVersion, expectDate string) {
			v := &Version{KubeBuilderVersion: initialVersion}

			v.applyVCSMetadata(settings)

			Expect(v.GitCommit).To(Equal(expectCommit))
			Expect(v.KubeBuilderVersion).To(Equal(expectVersion))
			Expect(v.BuildDate).To(Equal(expectDate))
		},
		Entry("Clean release build", testKubebuilderV410,
			[]debug.BuildSetting{
				{Key: vcsKeyRevision, Value: testGitCommit},
				{Key: vcsKeyTime, Value: "2025-12-27T18:00:00Z"},
				{Key: vcsKeyModified, Value: "false"},
			},
			testGitCommit, testKubebuilderV410, "2025-12-27T18:00:00Z"),
		Entry("Dirty development build", develVersion,
			[]debug.BuildSetting{
				{Key: vcsKeyRevision, Value: testGitCommit},
				{Key: vcsKeyModified, Value: vcsValueTrue},
				{Key: vcsKeyTime, Value: "2025-12-29T19:30:00Z"},
			},
			testGitCommitDirty, develVersion, "2025-12-29T19:30:00Z"),
		Entry("Dirty tagged release (GoReleaser scenario)", "v4.5.3-rc.1",
			[]debug.BuildSetting{
				{Key: vcsKeyRevision, Value: testGitCommit},
				{Key: vcsKeyModified, Value: vcsValueTrue},
				{Key: vcsKeyTime, Value: "2025-12-30T10:00:00Z"},
			},
			// Stays clean for tagged releases
			testGitCommitDirty, "v4.5.3-rc.1", "2025-12-30T10:00:00Z"),
		Entry("Dirty pseudo-version", "v1.2.4-0.20191109021931-daa7c04131f5",
			[]debug.BuildSetting{
				{Key: vcsKeyRevision, Value: testGitCommit},
				{Key: vcsKeyModified, Value: vcsValueTrue},
			},
			// Pseudo-versions become (devel) when dirty
			testGitCommitDirty, develVersion, ""),
	)

	It("should print every field of the version", func() {
		v := Version{
			KubeBuilderVersion: "v9.99.9",
			KubernetesVendor:   "9.99.9",
			GitCommit:          "9990f08847dd1",
			BuildDate:          "1970-01-12T12:12:12Z",
			GoOs:               "linux",
			GoArch:             "amd64",
		}

		expectedOutput := `KubeBuilder:          v9.99.9
Kubernetes:           9.99.9
Git Commit:           9990f08847dd1
Build Date:           1970-01-12T12:12:12Z
Go OS/Arch:           linux/amd64`

		Expect(v.PrintVersion()).To(Equal(expectedOutput))
	})
})
