package e2e

import (
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &SuiteTest{}

type SuiteTest struct {
	machinery.TemplateMixin
	machinery.RepositoryMixin
	machinery.ProjectNameMixin
}

func (f *SuiteTest) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = "test/e2e/e2e_suite_test.go"
	}

	f.TemplateBody = suiteTestTemplate
	return nil
}

var suiteTestTemplate = `package e2e

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Run e2e tests using the Ginkgo runner.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	fmt.Fprintf(GinkgoWriter, "Starting {{ .ProjectName }} suite\n")
	RunSpecs(t, "e2e suite")
}
`
