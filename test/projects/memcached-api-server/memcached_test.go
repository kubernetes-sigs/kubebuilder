package memcached_test

import (
	"github.com/kubernetes-sigs/kubebuilder/test/internal/e2e"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

var kubebuilderTest *e2e.KubebuilderTest

func setup() {
	_, filename, _, _ := runtime.Caller(0)
	projectDir := filepath.Dir(filename)
	kubebuilderBin := "/tmp/kubebuilder/bin"
	kubebuilderTest = e2e.NewKubebuilderTest(projectDir, kubebuilderBin)
}

func teardown() {
	kubebuilderTest.CleanUp()
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func TestGenerateBuildTest(t *testing.T) {
	err := kubebuilderTest.Generate([]string{"--skip-rbac-validation"})
	if err != nil {
		t.Errorf(err.Error())
	}
	err = kubebuilderTest.Build()
	if err != nil {
		t.Errorf(err.Error())
	}
	err = kubebuilderTest.Test()
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestDocs(t *testing.T) {
	// (droot): Disabling docs test for now because they are broken for k8s 1.1.
	// Fix them when we start supporting docs for v1 projects in KB
	t.Skip()
	docsOptions := []string{"--docs-copyright", "Hello", "--title", "World", "--cleanup=false", "--brodocs=false"}
	err := kubebuilderTest.Docs(docsOptions)
	if err != nil {
		t.Errorf(err.Error())
	}
	docsDir := filepath.Join(kubebuilderTest.Dir, "docs", "reference")
	expectedDocsDir := filepath.Join(kubebuilderTest.Dir, "test", "docs", "reference")
	err = kubebuilderTest.DiffAll(docsDir, expectedDocsDir)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestCreateConfig(t *testing.T) {
	configOptions := []string{"--crds"}
	err := kubebuilderTest.CreateConfig(configOptions)
	if err != nil {
		t.Errorf(err.Error())
	}
	configFile := filepath.Join(kubebuilderTest.Dir, "hack", "install.yaml")
	expectedConfigFile := filepath.Join(kubebuilderTest.Dir, "test", "hack", "install.yaml")
	err = kubebuilderTest.Diff(configFile, expectedConfigFile)
	if err != nil {
		t.Errorf(err.Error())
	}
}
