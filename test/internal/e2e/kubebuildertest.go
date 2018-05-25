package e2e

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type KubebuilderTest struct {
	Dir string
}

func NewKubebuilderTest(dir, binDir string) *KubebuilderTest {
	kt := KubebuilderTest{Dir: dir}
	os.Setenv("TEST_ASSET_KUBECTL", strings.Join([]string{binDir, "kubectl"}, "/"))
	os.Setenv("TEST_ASSET_KUBE_APISERVER", strings.Join([]string{binDir, "kube-apiserver"}, "/"))
	os.Setenv("TEST_ASSET_ETCD", strings.Join([]string{binDir, "etcd"}, "/"))
	cmd := exec.Command("command", "-v", "kubebuilder")
	if err := kt.runCommand(cmd); err != nil {
		os.Setenv("PATH",strings.Join([]string{binDir, os.Getenv("PATH")}, ":"))
	}
	return &kt
}

func (kt *KubebuilderTest) Generate(generateOptions []string) error {
	generateOptions = append([]string{"generate"}, generateOptions...)
	cmd := exec.Command("kubebuilder", generateOptions...)
	return kt.runCommand(cmd)
}

func (kt *KubebuilderTest) Docs(docsOptions []string) error {
	docsOptions = append([]string{"docs"}, docsOptions...)
	cmd := exec.Command("kubebuilder", docsOptions...)
	return kt.runCommand(cmd)
}

func (kt *KubebuilderTest) Build() error {
	var errs []string
	cmd := exec.Command("go", "build", "./pkg/...")
	err := kt.runCommand(cmd)
	if err != nil {
		errs = append(errs, err.Error())
	}
	cmd = exec.Command("go", "build", "./cmd/...")
	err = kt.runCommand(cmd)
	if err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}

func (kt *KubebuilderTest) Test() error {
	var errs []string
	cmd := exec.Command("go", "test", "./pkg/...")
	err := kt.runCommand(cmd)
	if err != nil {
		errs = append(errs, err.Error())
	}
	cmd = exec.Command("go", "test", "./cmd/...")
	err = kt.runCommand(cmd)
	if err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}

func (kt *KubebuilderTest) CreateConfig(configOptions []string) error {
	configOptions = append([]string{"create", "config"}, configOptions...)
	cmd := exec.Command("kubebuilder", configOptions...)
	return kt.runCommand(cmd)
}

func (kt *KubebuilderTest) Diff(pathA, pathB string) error {
	cmd := exec.Command("diff", pathA, pathB)
	return kt.runCommand(cmd)
}

func (kt *KubebuilderTest) DiffAll(generatedDir, expectedDir string) error {
	files, err := ioutil.ReadDir(expectedDir)
	if err != nil {
		return err
	}
	var errs []string
	for _, f := range files {
		generatedFile := filepath.Join(generatedDir, f.Name())
		if _, err := os.Stat(generatedFile); err != nil {
			errs = append(errs, err.Error())
		} else {
			err = kt.Diff(generatedFile, filepath.Join(expectedDir, f.Name()))
			if err != nil {
				errs = append(errs, err.Error())
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}

func (kt *KubebuilderTest) DepEnsure() error {
	cmd := exec.Command("dep", "ensure")
	return kt.runCommand(cmd)
}

func (kt *KubebuilderTest) VendorUpdate() error {
	cmd := exec.Command("kubebuilder", "vendor", "update")
	return kt.runCommand(cmd)
}

func (kt *KubebuilderTest) CleanUp() error {
	var errs []string
	cmd := exec.Command("kubebuilder", "generate", "clean")
	err := kt.runCommand(cmd)
	if err != nil {
		errs = append(errs, err.Error())
	}
	cmd = exec.Command("rm", "-r", "docs")
	err = kt.runCommand(cmd)
	if err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return fmt.Errorf(strings.Join(errs, "\n"))
	}
	return nil
}

func (kt *KubebuilderTest) runCommand(cmd *exec.Cmd) error {
	cmd.Dir = kt.Dir
	cmd.Env = os.Environ()
	command := strings.Join(cmd.Args, " ")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("%s failed with error: %s", command, string(output))
	}
	log.Printf("%s finished successfully", command)
	return nil
}
