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

package utils

import (
	"os"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// CheckError will exit with exit code 1 when err is not nil.
func CheckError(msg string, err error) {
	if err != nil {
		log.Errorf("error %s: %s", msg, err)
		os.Exit(1)
	}
}

// NewSampleContext return a context for the Sample
func NewSampleContext(binaryPath string, samplePath string, env ...string) utils.TestContext {
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
