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
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
)

var _ = Describe("Version", func() {
	// Parse, String and Validate are tested by MarshalJSON and UnmarshalJSON

	Context("Compare", func() {
		// Test Compare() by sorting a list.
		var (
			versions = []Version{
				{Number: 2, Stage: stage.Alpha},
				{Number: 44, Stage: stage.Alpha},
				{Number: 1},
				{Number: 2, Stage: stage.Beta},
				{Number: 4, Stage: stage.Beta},
				{Number: 1, Stage: stage.Alpha},
				{Number: 4},
				{Number: 44, Stage: stage.Alpha},
				{Number: 30},
				{Number: 4, Stage: stage.Alpha},
			}

			sortedVersions = []Version{
				{Number: 1, Stage: stage.Alpha},
				{Number: 1},
				{Number: 2, Stage: stage.Alpha},
				{Number: 2, Stage: stage.Beta},
				{Number: 4, Stage: stage.Alpha},
				{Number: 4, Stage: stage.Beta},
				{Number: 4},
				{Number: 30},
				{Number: 44, Stage: stage.Alpha},
				{Number: 44, Stage: stage.Alpha},
			}
		)

		It("sorts a valid list of versions correctly", func() {
			sort.Slice(versions, func(i int, j int) bool {
				return versions[i].Compare(versions[j]) == -1
			})
			Expect(versions).To(Equal(sortedVersions))
		})
	})

	Context("IsStable", func() {
		DescribeTable("should return true for stable versions",
			func(version Version) { Expect(version.IsStable()).To(BeTrue()) },
			Entry("for version 1", Version{Number: 1}),
			Entry("for version 1 (stable)", Version{Number: 1, Stage: stage.Stable}),
			Entry("for version 22", Version{Number: 22}),
			Entry("for version 22 (stable)", Version{Number: 22, Stage: stage.Stable}),
		)

		DescribeTable("should return false for unstable versions",
			func(version Version) { Expect(version.IsStable()).To(BeFalse()) },
			Entry("for version 1 (alpha)", Version{Number: 1, Stage: stage.Alpha}),
			Entry("for version 1 (beta)", Version{Number: 1, Stage: stage.Beta}),
			Entry("for version 22 (alpha)", Version{Number: 22, Stage: stage.Alpha}),
			Entry("for version 22 (beta)", Version{Number: 22, Stage: stage.Beta}),
		)
	})

	Context("MarshalJSON", func() {
		DescribeTable("should be marshalled appropriately",
			func(version Version, str string) {
				b, err := version.MarshalJSON()
				Expect(err).NotTo(HaveOccurred())
				Expect(string(b)).To(Equal(str))
			},
			Entry("for version 1", Version{Number: 1}, `"1"`),
			Entry("for version 1 (stable)", Version{Number: 1, Stage: stage.Stable}, `"1"`),
			Entry("for version 1 (alpha)", Version{Number: 1, Stage: stage.Alpha}, `"1-alpha"`),
			Entry("for version 1 (beta)", Version{Number: 1, Stage: stage.Beta}, `"1-beta"`),
			Entry("for version 22", Version{Number: 22}, `"22"`),
			Entry("for version 22 (stable)", Version{Number: 22, Stage: stage.Stable}, `"22"`),
			Entry("for version 22 (alpha)", Version{Number: 22, Stage: stage.Alpha}, `"22-alpha"`),
			Entry("for version 22 (beta)", Version{Number: 22, Stage: stage.Beta}, `"22-beta"`),
		)

		DescribeTable("should fail to be marshalled",
			func(version Version) {
				_, err := version.MarshalJSON()
				Expect(err).To(HaveOccurred())
			},
			Entry("for version 0", Version{Number: 0}),
			Entry("for version 0 (stable)", Version{Number: 0, Stage: stage.Stable}),
			Entry("for version 0 (alpha)", Version{Number: 0, Stage: stage.Alpha}),
			Entry("for version 0 (beta)", Version{Number: 0, Stage: stage.Beta}),
			Entry("for version 0 (implicit)", Version{}),
			Entry("for version 0 (stable) (implicit)", Version{Stage: stage.Stable}),
			Entry("for version 0 (alpha) (implicit)", Version{Stage: stage.Alpha}),
			Entry("for version 0 (beta) (implicit)", Version{Stage: stage.Beta}),
			Entry("for version -1", Version{Number: -1}),
			Entry("for version -1 (stable)", Version{Number: -1, Stage: stage.Stable}),
			Entry("for version -1 (alpha)", Version{Number: -1, Stage: stage.Alpha}),
			Entry("for version -1 (beta)", Version{Number: -1, Stage: stage.Beta}),
			Entry("for invalid stage", Version{Stage: stage.Stage(34)}),
		)
	})

	Context("UnmarshalJSON", func() {
		DescribeTable("should be unmarshalled appropriately",
			func(str string, number int, s stage.Stage) {
				var v Version
				err := v.UnmarshalJSON([]byte(str))
				Expect(err).NotTo(HaveOccurred())
				Expect(v.Number).To(Equal(number))
				Expect(v.Stage).To(Equal(s))
			},
			Entry("for version string `1`", `"1"`, 1, stage.Stable),
			Entry("for version string `1-alpha`", `"1-alpha"`, 1, stage.Alpha),
			Entry("for version string `1-beta`", `"1-beta"`, 1, stage.Beta),
			Entry("for version string `22`", `"22"`, 22, stage.Stable),
			Entry("for version string `22-alpha`", `"22-alpha"`, 22, stage.Alpha),
			Entry("for version string `22-beta`", `"22-beta"`, 22, stage.Beta),
		)

		DescribeTable("should fail to be unmarshalled",
			func(str string) {
				var v Version
				err := v.UnmarshalJSON([]byte(str))
				Expect(err).To(HaveOccurred())
			},
			Entry("for empty version string", ``),
			Entry("for version string ``", `""`),
			Entry("for version string `0`", `"0"`),
			Entry("for version string `0-alpha`", `"0-alpha"`),
			Entry("for version string `0-beta`", `"0-beta"`),
			Entry("for version string `-1`", `"-1"`),
			Entry("for version string `-1-alpha`", `"-1-alpha"`),
			Entry("for version string `-1-beta`", `"-1-beta"`),
			Entry("for version string `v1`", `"v1"`),
			Entry("for version string `v1-alpha`", `"v1-alpha"`),
			Entry("for version string `v1-beta`", `"v1-beta"`),
			Entry("for version string `1.0`", `"1.0"`),
			Entry("for version string `v1.0`", `"v1.0"`),
			Entry("for version string `v1.0-alpha`", `"v1.0-alpha"`),
			Entry("for version string `1.0.0`", `"1.0.0"`),
			Entry("for version string `1-a`", `"1-a"`),
		)
	})
})
