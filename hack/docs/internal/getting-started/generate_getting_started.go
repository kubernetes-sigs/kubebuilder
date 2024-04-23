/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gettingstarted

import (
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
)

type Sample struct {
	ctx *utils.TestContext
}

func NewSample(binaryPath, samplePath string) Sample {
	log.Infof("Generating the sample context of getting-started...")
	ctx := newSampleContext(binaryPath, samplePath, "GO111MODULE=on")
	return Sample{&ctx}
}

func (sp *Sample) UpdateTutorial() {
	log.Println("TODO: update tutorial")
}

func newSampleContext(binaryPath string, samplePath string, env ...string) utils.TestContext {
	cmdContext := &utils.CmdContext{
		Env: env,
		Dir: samplePath,
	}

	testContext := utils.TestContext{
		CmdContext: cmdContext,
		BinaryName: binaryPath,
	}

	return testContext
}

// Prepare the Context for the sample project
func (sp *Sample) Prepare() {
	log.Infof("Destroying directory for getting-started sample project")
	sp.ctx.Destroy()

	log.Infof("Refreshing tools and creating directory...")
	err := sp.ctx.Prepare()

	CheckError("Creating directory for sample project", err)
}

func (sp *Sample) GenerateSampleProject() {
	log.Infof("Initializing the getting started project")
	err := sp.ctx.Init(
		"--domain", "example.com",
		"--repo", "example.com/memcached",
		"--license", "apache2",
		"--owner", "The Kubernetes authors",
	)
	CheckError("Initializing the getting started project", err)

	log.Infof("Adding a new config type")
	err = sp.ctx.CreateAPI(
		"--group", "cache",
		"--version", "v1alpha1",
		"--kind", "Memcached",
		"--image", "memcached:1.4.36-alpine",
		"--image-container-command", "memcached,-m=64,-o,modern,-v",
		"--image-container-port", "11211",
		"--run-as-user", "1001",
		"--plugins", "deploy-image/v1-alpha",
		"--make=false",
	)
	CheckError("Creating the API", err)
}

func (sp *Sample) CodeGen() {
	cmd := exec.Command("make", "manifests")
	_, err := sp.ctx.Run(cmd)
	CheckError("Failed to run make manifests for getting started tutorial", err)

	cmd = exec.Command("make", "all")
	_, err = sp.ctx.Run(cmd)
	CheckError("Failed to run make all for getting started tutorial", err)

	cmd = exec.Command("go", "mod", "tidy")
	_, err = sp.ctx.Run(cmd)
	CheckError("Failed to run go mod tidy all for getting started tutorial", err)
}

// CheckError will exit with exit code 1 when err is not nil.
func CheckError(msg string, err error) {
	if err != nil {
		log.Errorf("error %s: %s", msg, err)
		os.Exit(1)
	}
}
