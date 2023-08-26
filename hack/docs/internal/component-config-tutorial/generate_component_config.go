/*
Copyright 2023 The Kubernetes Authors.

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

package componentconfig

import (
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/afero"
	pluginutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
)

type Sample struct {
	ctx *utils.TestContext
}

func NewSample(binaryPath, samplePath string) Sample {
	log.Infof("Generating the sample context of component-config...")

	ctx := newSampleContext(binaryPath, samplePath, "GO111MODULE=on")

	return Sample{&ctx}
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
	log.Infof("destroying directory for component_config sample project")
	sp.ctx.Destroy()

	log.Infof("refreshing tools and creating directory...")
	err := sp.ctx.Prepare()

	CheckError("creating directory for sample project", err)
}

func (sp *Sample) GenerateSampleProject() {
	log.Infof("Initializing the component config project")
	err := sp.ctx.Init(
		"--domain", "tutorial.kubebuilder.io",
		"--repo", "tutorial.kubebuilder.io/project",
		"--license", "apache2",
		"--owner", "The Kubernetes authors",
		"--plugins=go/v4",
		"--component-config",
	)
	CheckError("Initializing the component config project", err)

	log.Infof("Adding a new config type")
	err = sp.ctx.CreateAPI(
		"--group", "config",
		"--version", "v2",
		"--kind", "ProjectConfig",
		"--resource", "--controller=false",
		"--make=false",
	)
	CheckError("Creating the API", err)
}

func (sp *Sample) UpdateTutorial() {
	// 1. generate controller_manager_config.yaml
	var fs = afero.NewOsFs()
	err := afero.WriteFile(fs, filepath.Join(sp.ctx.Dir, "config/manager/controller_manager_config.yaml"),
		[]byte(`apiVersion: controller-runtime.sigs.k8s.io/v1alpha1
kind: ControllerManagerConfig
metadata:
  labels:
    app.kubernetes.io/name: controllermanagerconfig
    app.kubernetes.io/instance: controller-manager-configuration
    app.kubernetes.io/component: manager
    app.kubernetes.io/created-by: project
    app.kubernetes.io/part-of: project
    app.kubernetes.io/managed-by: kustomize
health:
  healthProbeBindAddress: :8081
metrics:
  bindAddress: 127.0.0.1:8080
webhook:
  port: 9443
leaderElection:
  leaderElect: true
  resourceName: 80807133.tutorial.kubebuilder.io
clusterName: example-test
`), 0600)
	CheckError("fixing controller_manager_config", err)

	// 2. fix projectconfig_types.go
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v2/projectconfig_types.go"),
		`metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"`,
		`
	cfg "sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"`)
	CheckError("fixing projectconfig_types", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "api/v2/projectconfig_types.go"),
		`Status ProjectConfigStatus `+"`"+`json:"status,omitempty"`+"`",
		`
	// ControllerManagerConfigurationSpec returns the configurations for controllers
	cfg.ControllerManagerConfigurationSpec `+"`"+`json:",inline"`+"`"+`

	ClusterName string `+"`"+`json:"clusterName,omitempty"`+"`",
	)

	CheckError("fixing projectconfig_types", err)

	// 3. fix main
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`var err error`,
		`
	ctrlConfig := configv2.ProjectConfig{}`)
	CheckError("fixing main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "cmd/main.go"),
		`AtPath(configFile)`,
		`.OfKind(&ctrlConfig)`)
	CheckError("fixing main.go", err)
}

func (sp *Sample) CodeGen() {

	cmd := exec.Command("make", "manifests")
	_, err := sp.ctx.Run(cmd)
	CheckError("Failed to run make manifests for componentConfig tutorial", err)

	cmd = exec.Command("make", "all")
	_, err = sp.ctx.Run(cmd)
	CheckError("Failed to run make all for componentConfig tutorial", err)

	cmd = exec.Command("go", "mod", "tidy")
	_, err = sp.ctx.Run(cmd)
	CheckError("Failed to run go mod tidy all for componentConfig tutorial", err)
}

// CheckError will exit with exit code 1 when err is not nil.
func CheckError(msg string, err error) {
	if err != nil {
		log.Errorf("error %s: %s", msg, err)
		os.Exit(1)
	}
}
