package project_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestProject(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Project Suite")
}
