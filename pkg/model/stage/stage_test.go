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

package stage

import (
	"sort"
	"testing"

	g "github.com/onsi/ginkgo" // An alias is required because Context is defined elsewhere in this package.
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func TestStage(t *testing.T) {
	RegisterFailHandler(g.Fail)
	g.RunSpecs(t, "Stage Suite")
}

var _ = g.Describe("ParseStage", func() {
	DescribeTable("should be correctly parsed for valid stage strings",
		func(str string, stage Stage) {
			s, err := ParseStage(str)
			Expect(err).NotTo(HaveOccurred())
			Expect(s).To(Equal(stage))
		},
		Entry("for alpha stage", "alpha", Alpha),
		Entry("for beta stage", "beta", Beta),
		Entry("for stable stage", "", Stable),
	)

	DescribeTable("should error when parsing invalid stage strings",
		func(str string) {
			_, err := ParseStage(str)
			Expect(err).To(HaveOccurred())
		},
		Entry("passing a number as the stage string", "1"),
		Entry("passing `gamma` as the stage string", "gamma"),
		Entry("passing a dash-prefixed stage string", "-alpha"),
	)
})

var _ = g.Describe("Stage", func() {
	g.Context("String", func() {
		DescribeTable("should return the correct string value",
			func(stage Stage, str string) { Expect(stage.String()).To(Equal(str)) },
			Entry("for alpha stage", Alpha, "alpha"),
			Entry("for beta stage", Beta, "beta"),
			Entry("for stable stage", Stable, ""),
		)

		DescribeTable("should panic",
			func(stage Stage) { Expect(func() { _ = stage.String() }).To(Panic()) },
			Entry("for stage 34", Stage(34)),
			Entry("for stage 75", Stage(75)),
			Entry("for stage 123", Stage(123)),
			Entry("for stage 255", Stage(255)),
		)
	})

	g.Context("Validate", func() {
		DescribeTable("should validate existing stages",
			func(stage Stage) { Expect(stage.Validate()).To(Succeed()) },
			Entry("for alpha stage", Alpha),
			Entry("for beta stage", Beta),
			Entry("for stable stage", Stable),
		)

		DescribeTable("should fail for non-existing stages",
			func(stage Stage) { Expect(stage.Validate()).NotTo(Succeed()) },
			Entry("for stage 34", Stage(34)),
			Entry("for stage 75", Stage(75)),
			Entry("for stage 123", Stage(123)),
			Entry("for stage 255", Stage(255)),
		)
	})

	g.Context("Compare", func() {
		// Test Stage.Compare by sorting a list
		var (
			stages = []Stage{
				Stable,
				Alpha,
				Stable,
				Beta,
				Beta,
				Alpha,
			}

			sortedStages = []Stage{
				Alpha,
				Alpha,
				Beta,
				Beta,
				Stable,
				Stable,
			}
		)

		g.It("sorts stages correctly", func() {
			sort.Slice(stages, func(i int, j int) bool {
				return stages[i].Compare(stages[j]) == -1
			})
			Expect(stages).To(Equal(sortedStages))
		})
	})

	g.Context("IsStable", func() {
		g.It("should return true for stable stage", func() {
			Expect(Stable.IsStable()).To(BeTrue())
		})

		DescribeTable("should return false for any unstable stage",
			func(stage Stage) { Expect(stage.IsStable()).To(BeFalse()) },
			Entry("for alpha stage", Alpha),
			Entry("for beta stage", Beta),
		)
	})
})
