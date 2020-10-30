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

package plugin

import (
	"sort"
	"testing"

	g "github.com/onsi/ginkgo" // An alias is required because Context is defined elsewhere in this package.
	. "github.com/onsi/gomega"
)

func TestCLI(t *testing.T) {
	RegisterFailHandler(g.Fail)
	g.RunSpecs(t, "Plugin Suite")
}

var _ = g.Describe("ParseStage", func() {
	var (
		s   Stage
		err error
	)

	g.It("should be correctly parsed for valid stage strings", func() {
		g.By("passing an empty stage string")
		s, err = ParseStage("")
		Expect(err).NotTo(HaveOccurred())
		Expect(s).To(Equal(StableStage))

		g.By("passing `alpha` as the stage string")
		s, err = ParseStage("alpha")
		Expect(err).NotTo(HaveOccurred())
		Expect(s).To(Equal(AlphaStage))

		g.By("passing `beta` as the stage string")
		s, err = ParseStage("beta")
		Expect(err).NotTo(HaveOccurred())
		Expect(s).To(Equal(BetaStage))
	})

	g.It("should error when parsing invalid stage strings", func() {
		g.By("passing a number as the stage string")
		s, err = ParseStage("1")
		Expect(err).To(HaveOccurred())

		g.By("passing `gamma` as the stage string")
		s, err = ParseStage("gamma")
		Expect(err).To(HaveOccurred())

		g.By("passing a dash-prefixed stage string")
		s, err = ParseStage("-alpha")
		Expect(err).To(HaveOccurred())
	})
})

var _ = g.Describe("Stage.String", func() {
	g.It("should return the correct string value", func() {
		g.By("for stable stage")
		Expect(StableStage.String()).To(Equal(stableStage))

		g.By("for alpha stage")
		Expect(AlphaStage.String()).To(Equal(alphaStage))

		g.By("for beta stage")
		Expect(BetaStage.String()).To(Equal(betaStage))
	})
})

var _ = g.Describe("Stage.Validate", func() {
	g.It("should validate existing stages", func() {
		g.By("for stable stage")
		Expect(StableStage.Validate()).NotTo(HaveOccurred())

		g.By("for alpha stage")
		Expect(AlphaStage.Validate()).NotTo(HaveOccurred())

		g.By("for beta stage")
		Expect(BetaStage.Validate()).NotTo(HaveOccurred())
	})

	g.It("should fail for non-existing stages", func() {
		Expect(Stage(34).Validate()).To(HaveOccurred())
		Expect(Stage(75).Validate()).To(HaveOccurred())
		Expect(Stage(123).Validate()).To(HaveOccurred())
		Expect(Stage(255).Validate()).To(HaveOccurred())
	})
})

var _ = g.Describe("Stage.Compare", func() {
	// Test Stage.Compare by sorting a list
	var (
		stages = []Stage{
			StableStage,
			BetaStage,
			AlphaStage,
		}

		sortedStages = []Stage{
			AlphaStage,
			BetaStage,
			StableStage,
		}
	)
	g.It("sorts stages correctly", func() {
		sort.Slice(stages, func(i int, j int) bool {
			return stages[i].Compare(stages[j]) == -1
		})
		Expect(stages).To(Equal(sortedStages))
	})
})

var _ = g.Describe("ParseVersion", func() {
	var (
		v   Version
		err error
	)

	g.It("should be correctly parsed when a version is positive without a stage", func() {
		g.By("passing version string 1")
		v, err = ParseVersion("1")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(1)))
		Expect(v.Stage).To(Equal(StableStage))

		g.By("passing version string 22")
		v, err = ParseVersion("22")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(22)))
		Expect(v.Stage).To(Equal(StableStage))

		g.By("passing version string v1")
		v, err = ParseVersion("v1")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(1)))
		Expect(v.Stage).To(Equal(StableStage))
	})

	g.It("should be correctly parsed when a version is positive with a stage", func() {
		g.By("passing version string 1-alpha")
		v, err = ParseVersion("1-alpha")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(1)))
		Expect(v.Stage).To(Equal(AlphaStage))

		g.By("passing version string 1-beta")
		v, err = ParseVersion("1-beta")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(1)))
		Expect(v.Stage).To(Equal(BetaStage))

		g.By("passing version string v1-alpha")
		v, err = ParseVersion("v1-alpha")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(1)))
		Expect(v.Stage).To(Equal(AlphaStage))

		g.By("passing version string v22-alpha")
		v, err = ParseVersion("v22-alpha")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(22)))
		Expect(v.Stage).To(Equal(AlphaStage))
	})

	g.It("should fail for invalid version strings", func() {
		g.By("passing version string an empty string")
		_, err = ParseVersion("")
		Expect(err).To(HaveOccurred())

		g.By("passing version string 0")
		_, err = ParseVersion("0")
		Expect(err).To(HaveOccurred())

		g.By("passing a negative number version string")
		_, err = ParseVersion("-1")
		Expect(err).To(HaveOccurred())
	})

	g.It("should fail validation when a version string is semver", func() {
		g.By("passing version string 1.0")
		_, err = ParseVersion("1.0")
		Expect(err).To(HaveOccurred())

		g.By("passing version string v1.0")
		_, err = ParseVersion("v1.0")
		Expect(err).To(HaveOccurred())

		g.By("passing version string v1.0-alpha")
		_, err = ParseVersion("v1.0-alpha")
		Expect(err).To(HaveOccurred())

		g.By("passing version string 1.0.0")
		_, err = ParseVersion("1.0.0")
		Expect(err).To(HaveOccurred())
	})
})

var _ = g.Describe("Version.String", func() {
	g.It("should return the correct string value", func() {
		g.By("for stable version 1")
		Expect(Version{Number: 1}.String()).To(Equal("v1"))
		Expect(Version{Number: 1, Stage: StableStage}.String()).To(Equal("v1"))

		g.By("for stable version 22")
		Expect(Version{Number: 22}.String()).To(Equal("v22"))
		Expect(Version{Number: 22, Stage: StableStage}.String()).To(Equal("v22"))

		g.By("for alpha version 1")
		Expect(Version{Number: 1, Stage: AlphaStage}.String()).To(Equal("v1-alpha"))

		g.By("for beta version 1")
		Expect(Version{Number: 1, Stage: BetaStage}.String()).To(Equal("v1-beta"))

		g.By("for alpha version 22")
		Expect(Version{Number: 22, Stage: AlphaStage}.String()).To(Equal("v22-alpha"))
	})
})

var _ = g.Describe("Stage.Validate", func() {
	g.It("should success for a positive version without a stage", func() {
		g.By("for version 1")
		Expect(Version{Number: 1}.Validate()).NotTo(HaveOccurred())

		g.By("for version 22")
		Expect(Version{Number: 22}.Validate()).NotTo(HaveOccurred())
	})

	g.It("should success for a positive version with a stage", func() {
		g.By("for version 1 alpha")
		Expect(Version{Number: 1, Stage: AlphaStage}.Validate()).NotTo(HaveOccurred())

		g.By("for version 1 beta")
		Expect(Version{Number: 1, Stage: BetaStage}.Validate()).NotTo(HaveOccurred())

		g.By("for version 22 alpha")
		Expect(Version{Number: 22, Stage: AlphaStage}.Validate()).NotTo(HaveOccurred())
	})

	g.It("should fail for invalid versions", func() {
		g.By("passing version 0")
		Expect(Version{}.Validate()).To(HaveOccurred())
		Expect(Version{Number: 0}.Validate()).To(HaveOccurred())

		g.By("passing a negative version")
		Expect(Version{Number: -1}.Validate()).To(HaveOccurred())

		g.By("passing an invalid stage")
		Expect(Version{Number: 1, Stage: Stage(173)}.Validate()).To(HaveOccurred())
	})
})

var _ = g.Describe("Version.Compare", func() {
	// Test Compare() by sorting a list.
	var (
		versions = []Version{
			{Number: 2, Stage: AlphaStage},
			{Number: 44, Stage: AlphaStage},
			{Number: 1},
			{Number: 2, Stage: BetaStage},
			{Number: 4, Stage: BetaStage},
			{Number: 1, Stage: AlphaStage},
			{Number: 4},
			{Number: 44, Stage: AlphaStage},
			{Number: 30},
			{Number: 4, Stage: AlphaStage},
		}

		sortedVersions = []Version{
			{Number: 1, Stage: AlphaStage},
			{Number: 1},
			{Number: 2, Stage: AlphaStage},
			{Number: 2, Stage: BetaStage},
			{Number: 4, Stage: AlphaStage},
			{Number: 4, Stage: BetaStage},
			{Number: 4},
			{Number: 30},
			{Number: 44, Stage: AlphaStage},
			{Number: 44, Stage: AlphaStage},
		}
	)

	g.It("sorts a valid list of versions correctly", func() {
		sort.Slice(versions, func(i int, j int) bool {
			return versions[i].Compare(versions[j]) == -1
		})
		Expect(versions).To(Equal(sortedVersions))
	})

})
