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

package cli

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	golangv4 "sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4"
)

// Two external resources sharing group, version and kind, told apart only by their domain.
// They are kept apart so the same pair can be laid out in either order.
const (
	certManagerIOEntry = `- domain: cert-manager.io
  external: true
  group: cert-manager
  kind: Issuer
  path: github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1
  version: v1
  webhooks:
    defaulting: true
    webhookVersion: v1
`
	certManagerK8sIOEntry = `- controller: true
  domain: cert-manager.k8s.io
  external: true
  group: cert-manager
  kind: Issuer
  path: github.com/other/cert-manager/apis/certmanager/v1
  version: v1
`
)

// Every command below is refused while the configuration is still being prepared, so nothing is
// ever scaffolded and no post-scaffold hook runs.
var _ = Describe("an ambiguous group, version and kind", func() {
	var (
		fs      machinery.Filesystem
		project string
	)

	// newCLI builds a CLI over an in-memory PROJECT holding the given entries in the given order.
	newCLI := func(entries ...string) *CLI {
		project = "domain: test.io\nlayout:\n- base.go.kubebuilder.io/v4\nprojectName: ambiguous\n" +
			"repo: github.com/example/ambiguous\nresources:\n"
		for _, e := range entries {
			project += e
		}
		project += "version: \"3\"\n"

		fs = machinery.Filesystem{FS: afero.NewMemMapFs()}
		Expect(afero.WriteFile(fs.FS, "PROJECT", []byte(project), 0o644)).To(Succeed())

		args := os.Args
		DeferCleanup(func() { os.Args = args })
		os.Args = []string{"kubebuilder"}

		c, err := New(
			WithPlugins(golangv4.Plugin{}),
			WithDefaultPlugins(cfgv3.Version, golangv4.Plugin{}),
			WithFilesystem(fs),
		)
		Expect(err).NotTo(HaveOccurred())
		return c
	}

	projectOnDisk := func() string {
		content, err := afero.ReadFile(fs.FS, "PROJECT")
		Expect(err).NotTo(HaveOccurred())
		return string(content)
	}

	webhookArgs := func(extra ...string) []string {
		return append([]string{
			"create", "webhook",
			groupFlagArg, "cert-manager", versionFlagArg, "v1", kindFlagArg, "Issuer", "--defaulting",
		}, extra...)
	}

	DescribeTable("is refused whatever order the project file lists the resources in",
		func(entries ...string) {
			c := newCLI(entries...)
			c.cmd.SetArgs(webhookArgs())

			err := c.cmd.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("match more than one resource"))
			Expect(err.Error()).To(ContainSubstring("cert-manager.cert-manager.io"))
			Expect(err.Error()).To(ContainSubstring("cert-manager.cert-manager.k8s.io"))
			Expect(err.Error()).To(ContainSubstring("--domain"))

			// Both entries keep their path, domain and webhooks: nothing was written.
			Expect(projectOnDisk()).To(Equal(project))
		},
		Entry("cert-manager.io first", certManagerIOEntry, certManagerK8sIOEntry),
		Entry("cert-manager.k8s.io first", certManagerK8sIOEntry, certManagerIOEntry),
	)

	// Only the cert-manager.io entry already has a defaulting webhook, so being told it exists
	// proves the command reached that resource rather than its cert-manager.k8s.io sibling.
	DescribeTable("is resolved to the resource the domain names",
		func(flag string) {
			c := newCLI(certManagerIOEntry, certManagerK8sIOEntry)
			c.cmd.SetArgs(webhookArgs(flag, "cert-manager.io"))

			err := c.cmd.Execute()

			Expect(err).To(MatchError(ContainSubstring("defaulting webhook already exists")))
			Expect(projectOnDisk()).To(Equal(project))
		},
		Entry("by --domain", "--domain"),
		Entry("by the deprecated --external-api-domain alias", "--external-api-domain"),
	)

	It("is not resolved by two disagreeing domain flags", func() {
		c := newCLI(certManagerIOEntry, certManagerK8sIOEntry)
		c.cmd.SetArgs(webhookArgs("--domain", "cert-manager.io", "--external-api-domain", "cert-manager.k8s.io"))

		err := c.cmd.Execute()

		Expect(err).To(MatchError(ContainSubstring("conflicting values")))
		Expect(projectOnDisk()).To(Equal(project))
	})
})
