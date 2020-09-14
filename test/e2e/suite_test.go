package e2e_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/test/e2e/utils"
)

var kbc *utils.TestContext

func TestE2e(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E Suite testing in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func(done Done) {
	var err error
	kbc, err = utils.NewTestContext(utils.KubebuilderBinName, "GO111MODULE=on")
	Expect(err).NotTo(HaveOccurred())

	By("installing cert manager bundle")
	Expect(kbc.InstallCertManager()).To(Succeed())

	By("installing prometheus operator")
	Expect(kbc.InstallPrometheusOperManager()).To(Succeed())

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("uninstalling prometheus manager bundle")
	kbc.UninstallPrometheusOperManager()
})
