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

var _ = g.Describe("ParseVersion", func() {

	var (
		v   Version
		err error
	)

	g.It("should pass validation when a version is positive without a stage", func() {
		g.By("passing version string 1")
		v, err = ParseVersion("1")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(1)))
		Expect(v.Stage).To(Equal(""))

		g.By("passing version string 22")
		v, err = ParseVersion("22")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(22)))
		Expect(v.Stage).To(Equal(""))

		g.By("passing version string v1")
		v, err = ParseVersion("v1")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(1)))
		Expect(v.Stage).To(Equal(""))
	})

	g.It("should pass validation when a version is positive with a stage", func() {
		g.By("passing version string 1-alpha")
		v, err = ParseVersion("1-alpha")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(1)))
		Expect(v.Stage).To(Equal("alpha"))

		g.By("passing version string 1-beta")
		v, err = ParseVersion("1-beta")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(1)))
		Expect(v.Stage).To(Equal("beta"))

		g.By("passing version string v1-alpha")
		v, err = ParseVersion("v1-alpha")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(1)))
		Expect(v.Stage).To(Equal("alpha"))

		g.By("passing version string v22-alpha")
		v, err = ParseVersion("v22-alpha")
		Expect(err).NotTo(HaveOccurred())
		Expect(v.Number).To(BeNumerically("==", int64(22)))
		Expect(v.Stage).To(Equal("alpha"))
	})

	g.It("should fail validation", func() {
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

var _ = g.Describe("Compare", func() {

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
			{Number: 1},
			{Number: 1, Stage: AlphaStage},
			{Number: 2, Stage: AlphaStage},
			{Number: 2, Stage: BetaStage},
			{Number: 4},
			{Number: 4, Stage: AlphaStage},
			{Number: 4, Stage: BetaStage},
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
