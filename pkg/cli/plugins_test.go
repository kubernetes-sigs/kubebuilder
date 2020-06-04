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

package cli

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

var _ = Describe("resolvePluginsByKey", func() {

	var (
		plugins = makePluginsForKeys(
			"foo.example.com/v1",
			"bar.example.com/v1",
			"baz.example.com/v1",
			"foo.kubebuilder.io/v1",
			"foo.kubebuilder.io/v2",
			"bar.kubebuilder.io/v1",
			"bar.kubebuilder.io/v2",
		)
		resolvedPlugins []plugin.Base
		err             error
	)

	It("should resolve keys correctly", func() {
		By("resolving foo.example.com/v1")
		resolvedPlugins, err = resolvePluginsByKey(plugins, "foo.example.com/v1")
		Expect(err).NotTo(HaveOccurred())
		Expect(makePluginKeySlice(resolvedPlugins...)).To(Equal([]string{"foo.example.com/v1"}))

		By("resolving foo.example.com")
		resolvedPlugins, err = resolvePluginsByKey(plugins, "foo.example.com")
		Expect(err).NotTo(HaveOccurred())
		Expect(makePluginKeySlice(resolvedPlugins...)).To(Equal([]string{"foo.example.com/v1"}))

		By("resolving baz")
		resolvedPlugins, err = resolvePluginsByKey(plugins, "baz")
		Expect(err).NotTo(HaveOccurred())
		Expect(makePluginKeySlice(resolvedPlugins...)).To(Equal([]string{"baz.example.com/v1"}))

		By("resolving foo/v2")
		resolvedPlugins, err = resolvePluginsByKey(plugins, "foo/v2")
		Expect(err).NotTo(HaveOccurred())
		Expect(makePluginKeySlice(resolvedPlugins...)).To(Equal([]string{"foo.kubebuilder.io/v2"}))
	})

	It("should return an error", func() {
		By("resolving foo.kubebuilder.io")
		_, err = resolvePluginsByKey(plugins, "foo.kubebuilder.io")
		Expect(err).To(MatchError(errAmbiguousPlugin{
			key: "foo.kubebuilder.io",
			msg: `matching plugins: ["foo.kubebuilder.io/v1" "foo.kubebuilder.io/v2"]`,
		}))

		By("resolving foo/v1")
		_, err = resolvePluginsByKey(plugins, "foo/v1")
		Expect(err).To(MatchError(errAmbiguousPlugin{
			key: "foo/v1",
			msg: `matching plugins: ["foo.example.com/v1" "foo.kubebuilder.io/v1"]`,
		}))

		By("resolving foo")
		_, err = resolvePluginsByKey(plugins, "foo")
		Expect(err).To(MatchError(errAmbiguousPlugin{
			key: "foo",
			msg: `matching plugins: ["foo.example.com/v1" "foo.kubebuilder.io/v1" "foo.kubebuilder.io/v2"]`,
		}))

		By("resolving blah")
		_, err = resolvePluginsByKey(plugins, "blah")
		Expect(err).To(MatchError(errAmbiguousPlugin{
			key: "blah",
			msg: fmt.Sprintf("no names match, possible plugins: %+q", makePluginKeySlice(plugins...)),
		}))

		By("resolving foo.example.com/v2")
		_, err = resolvePluginsByKey(plugins, "foo.example.com/v2")
		Expect(err).To(MatchError(errAmbiguousPlugin{
			key: "foo.example.com/v2",
			msg: fmt.Sprintf(`no versions match, possible plugins: ["foo.example.com/v1"]`),
		}))

		By("resolving foo/v3")
		_, err = resolvePluginsByKey(plugins, "foo/v3")
		Expect(err).To(MatchError(errAmbiguousPlugin{
			key: "foo/v3",
			msg: "no versions match, possible plugins: " +
				`["foo.example.com/v1" "foo.kubebuilder.io/v1" "foo.kubebuilder.io/v2"]`,
		}))

		By("resolving foo.example.com/v3")
		_, err = resolvePluginsByKey(plugins, "foo.example.com/v3")
		Expect(err).To(MatchError(errAmbiguousPlugin{
			key: "foo.example.com/v3",
			msg: `no versions match, possible plugins: ["foo.example.com/v1"]`,
		}))
	})
})
