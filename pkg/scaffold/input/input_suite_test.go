package input_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestInput(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Input Suite")
}
