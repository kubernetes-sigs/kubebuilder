package scaffold_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestScaffold(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scaffold Suite")
}
