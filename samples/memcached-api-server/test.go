package main

import (
	"log"
	"runtime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	projectDir := filepath.Dir(filename)
    runProjectTest(projectDir)
}

func runProjectTest(dir string) {
	os.Setenv("TEST_ASSET_KUBECTL", "/tmp/kubebuilder/bin/kubectl")
	os.Setenv("TEST_ASSET_KUBE_APISERVER", "/tmp/kubebuilder/bin/kube-apiserver")
	os.Setenv("TEST_ASSET_ETCD", "/tmp/kubebuilder/bin/etcd")

	log.Printf("Testing the project %s", dir)
	cmds := []*exec.Cmd{
		exec.Command("dep", "ensure"),
		exec.Command("kubebuilder", "generate"),
		exec.Command("kubebuilder", "docs", "--docs-copyright", "Hello", "--title", "World", "--cleanup=false", "--brodocs=false"),
		exec.Command("diff", "test/docs/reference/includes", "docs/reference/includes"),
		exec.Command("diff", "test/docs/reference/config.yaml", "docs/reference/config.yaml"),
		exec.Command("diff", "test/docs/reference/manifest.json", "docs/reference/manifest.json"),
		exec.Command("go", "build", "./pkg/..."),
		exec.Command("go", "build", "./cmd/..."),
		exec.Command("go", "test", "./pkg/..."),
		exec.Command("go", "test", "./cmd/..."),
        exec.Command("kubebuilder", "create", "config", "--crds"),
        exec.Command("diff", "test/hack/install.yaml", "hack/install.yaml"),
	}

    for _, c := range cmds {
    	c.Dir = dir
		c.Env = os.Environ()
    	command := strings.Join(c.Args, " ")
    	log.Printf("Running the command %s", command)
    	output, err := c.Output()
    	if err != nil {
			log.Fatalf("%s finished with error: %s", command, string(output))
		}
		log.Printf("%s finished successfully", command)
	}
}