package validations_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"github.com/kubernetes-sigs/kubebuilder/test/internal/e2e"
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
