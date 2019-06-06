package e2e

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
)

// kubebuilderTest contains context to run commands for kubebuilder
type kubebuilderTest struct {
	Dir string
	// environment variables in k=v format.
	Env []string
}

// Init is for running `kubebuilder init`
func (kt *kubebuilderTest) Init(initOptions ...string) error {
	initOptions = append([]string{"init"}, initOptions...)
	cmd := exec.Command("kubebuilder", initOptions...)
	_, err := kt.runCommand(cmd)
	return err
}

// CreateAPI is for running `kubebuilder create api`
func (kt *kubebuilderTest) CreateAPI(resourceOptions ...string) error {
	resourceOptions = append([]string{"create", "api"}, resourceOptions...)
	cmd := exec.Command("kubebuilder", resourceOptions...)
	_, err := kt.runCommand(cmd)
	return err
}

// Make is for running `make` with various targets
func (kt *kubebuilderTest) Make(makeOptions ...string) error {
	cmd := exec.Command("make", makeOptions...)
	_, err := kt.runCommand(cmd)
	return err
}

// CleanupImage is for cleaning up the docker images for testing
func (kt *kubebuilderTest) CleanupImage(imageOptions ...string) error {
	imageOptions = append([]string{"rmi", "-f"}, imageOptions...)
	cmd := exec.Command("docker", imageOptions...)
	_, err := kt.runCommand(cmd)
	return err
}

// RunKubectlCommand is a general func to run kubectl commands
func (kt *kubebuilderTest) RunKubectlCommand(cmdOptions ...string) (string, error) {
	cmd := exec.Command("kubectl", cmdOptions...)
	output, err := kt.runCommand(cmd)
	return string(output), err
}

// RunKubectlCommandWithInput is a general func to run kubectl commands with input
func (kt *kubebuilderTest) RunKubectlCommandWithInput(stdinInput string, cmdOptions ...string) (string, error) {
	cmd := exec.Command("kubectl", cmdOptions...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, stdinInput)
	}()
	output, err := kt.runCommand(cmd)
	return string(output), err
}

// RunKubectlGetPodsInNamespace is a func to run kubectl get pods -n <namepsace> commands
func (kt *kubebuilderTest) RunKubectlGetPodsInNamespace(testSuffix string, cmdOptions ...string) (string, error) {
	getPodsOptions := []string{"get", "pods", "-n", fmt.Sprintf("e2e-%s-system", testSuffix)}
	return kt.RunKubectlCommand(append(getPodsOptions, cmdOptions...)...)
}

// LoadImageToKindCluster loads a local docker image to the kind cluster
func (kt *kubebuilderTest) LoadImageToKindCluster(imageName string) error {
	kindOptions := []string{"load", "docker-image", imageName}
	cmd := exec.Command("kind", kindOptions...)
	_, err := kt.runCommand(cmd)
	return err
}

// RunKustomizeCommand is a general func to run kustomize commands
func (kt *kubebuilderTest) RunKustomizeCommand(kustomizeOptions ...string) (string, error) {
	cmd := exec.Command("kustomize", kustomizeOptions...)
	output, err := kt.runCommand(cmd)
	return string(output), err
}

func (kt *kubebuilderTest) runCommand(cmd *exec.Cmd) ([]byte, error) {
	cmd.Dir = kt.Dir
	cmd.Env = append(os.Environ(), kt.Env...)
	command := strings.Join(cmd.Args, " ")
	fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: %s", command, string(output))
	}

	return output, nil
}
