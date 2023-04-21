package util

import (
	"bytes"
	"strings"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestExecPlugin(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Plugin Util Suite")
}

var _ = ginkgo.Describe("RunCmd", func() {
	var (
		output *bytes.Buffer
		err    error
	)

	ginkgo.BeforeEach(func() {
		output = &bytes.Buffer{}
	})

	ginkgo.AfterEach(func() {
		output.Reset()
	})

	ginkgo.It("executes the command and redirects output to stdout", func() {
		err = RunCmd("echo test", "echo", "test")
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		gomega.Expect(strings.TrimSpace(output.String())).To(gomega.Equal("test"))
	})

	ginkgo.It("returns an error if the command fails", func() {
		err = RunCmd("unknown command", "unknowncommand")
		gomega.Expect(err).To(gomega.HaveOccurred())
	})
})
