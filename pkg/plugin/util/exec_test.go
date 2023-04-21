package util

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var _ = Describe("RunCmd", func() {
	var (
		output *bytes.Buffer
		err    error
	)

	BeforeEach(func() {
		output = &bytes.Buffer{}
	})

	AfterEach(func() {
		output.Reset()
	})

	It("executes the command and redirects output to stdout", func() {
		err = RunCmd("echo test", "echo", "test")
		Expect(err).ToNot(HaveOccurred())

		Expect(strings.TrimSpace(output.String())).To(Equal("test"))
	})

	It("returns an error if the command fails", func() {
		err = RunCmd("unknown command", "unknowncommand")
		Expect(err).To(HaveOccurred())
	})
})
