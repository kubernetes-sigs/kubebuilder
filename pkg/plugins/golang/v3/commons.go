/*
Copyright 2022 The Kubernetes Authors.

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

//go:deprecated This package has been deprecated in favor of v4
package v3

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds"
)

const deprecateMsg = "The v1beta1 API version for CRDs and Webhooks are deprecated and are no longer supported since " +
	"the Kubernetes release 1.22. This flag no longer required to exist in future releases. Also, we would like to " +
	"recommend you no longer use these API versions." +
	"More info: https://kubernetes.io/docs/reference/using-api/deprecation-guide/#v1-22"

// Update the makefile to allow generate CRDs/Webhooks with v1beta1 to ensure backwards compatibility
// nolint:lll,gosec
func applyScaffoldCustomizationsForVbeta1() error {
	makefilePath := filepath.Join("Makefile")
	bs, err := os.ReadFile(makefilePath)
	if err != nil {
		return err
	}
	if !strings.Contains(string(bs), "CRD_OPTIONS") {

		log.Warn("The v1beta1 API version for CRDs and Webhooks are deprecated and are no longer supported " +
			"since the Kubernetes release 1.22. In order to help you out use these versions" +
			"we will need to try to update the Makefile and go.mod files of this project. Please," +
			"ensure that these changes were done accordingly with your customizations.\n" +
			"Also, we would like to recommend you no longer use these API versions." +
			"More info: https://kubernetes.io/docs/reference/using-api/deprecation-guide/#v1-22")

		const makefileTarget = `$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases`
		const makefileTargetForV1beta1 = `$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases`

		if err := util.ReplaceInFile("Makefile", makefileTarget, makefileTargetForV1beta1); err != nil {
			fmt.Printf("unable to update the makefile to allow the usage of v1beta1: %s", err)
		}

		const makegentarget = `manifests: controller-gen`
		const makegenV1beta1Options = `# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:crdVersions={v1beta1},trivialVersions=true,preserveUnknownFields=false"
manifests: controller-gen`

		if err := util.ReplaceInFile("Makefile", makegentarget, makegenV1beta1Options); err != nil {
			log.Warnf("unable to update the Makefile with %s: %s", makegenV1beta1Options, err)
		}

		// latest version of controller-tools where v1beta1 is supported
		const controllerToolsVersionForVBeta1 = "v0.6.2"
		if err := util.ReplaceInFile("Makefile",
			fmt.Sprintf("CONTROLLER_TOOLS_VERSION ?= %s",
				scaffolds.ControllerToolsVersion),
			fmt.Sprintf("CONTROLLER_TOOLS_VERSION ?= %s",
				controllerToolsVersionForVBeta1)); err != nil {
			log.Warnf("unable to update the Makefile with %s: %s", fmt.Sprintf("controller-gen@%s",
				controllerToolsVersionForVBeta1), err)
		}

		if err := util.ReplaceInFile("Makefile",
			"ENVTEST_K8S_VERSION = 1.26.1",
			"ENVTEST_K8S_VERSION = 1.21"); err != nil {
			log.Warnf("unable to update the Makefile with %s: %s", "ENVTEST_K8S_VERSION = 1.21", err)
		}

		// DO NOT UPDATE THIS VERSION
		// Note that this implementation will update the go.mod to downgrade the versions for those that are
		// compatible v1beta1 CRD/Webhooks k8s core APIs if/when a user tries to create an API with
		// create api [options] crd-version=v1beta1. The flag/feature is deprecated. however, to ensure that backwards
		// compatible we must introduce this logic. Also, note that when we bump the k8s dependencies we need to
		// ensure that the following replacements will be done accordingly to downgrade the versions.
		// The next version of the Golang base plugin (go/v4) no longer provide this feature.
		const controllerRuntimeVersionForVBeta1 = "v0.9.2"

		if err := util.ReplaceInFile("go.mod",
			fmt.Sprintf("sigs.k8s.io/controller-runtime %s", scaffolds.ControllerRuntimeVersion),
			fmt.Sprintf("sigs.k8s.io/controller-runtime %s", controllerRuntimeVersionForVBeta1)); err != nil {
			log.Warnf("unable to update the go.mod with sigs.k8s.io/controller-runtime %s: %s",
				controllerRuntimeVersionForVBeta1, err)
		}

		if err := util.ReplaceInFile("go.mod",
			"k8s.io/api v0.24.2",
			"k8s.io/api v0.21.2"); err != nil {
			log.Warnf("unable to update the go.mod with k8s.io/api v0.21.2: %s", err)
		}

		if err := util.ReplaceInFile("go.mod",
			"k8s.io/apimachinery v0.24.2",
			"k8s.io/apimachinery v0.21.2"); err != nil {
			log.Warnf("unable to update the go.mod with k8s.io/apimachinery v0.21.2: %s", err)
		}

		if err := util.ReplaceInFile("go.mod",
			"k8s.io/client-go v0.24.2",
			"k8s.io/client-go v0.21.2"); err != nil {
			log.Warnf("unable to update the go.mod with k8s.io/client-go v0.21.2: %s", err)
		}

		// During the scaffolding phase, this gets added to go.mod file, running go mod tidy bumps back
		// the version from 21.2 to the latest
		if err := util.ReplaceInFile("go.mod",
			"k8s.io/api v0.24.2",
			"k8s.io/api v0.21.2"); err != nil {
			log.Warnf("unable to update the go.mod with k8s.io/api v0.21.2: %s", err)
		}

		if err := util.ReplaceInFile("go.mod",
			"k8s.io/apiextensions-apiserver v0.24.2",
			"k8s.io/apiextensions-apiserver v0.21.2"); err != nil {
			log.Warnf("unable to update the go.mod with k8s.io/apiextensions-apiserver v0.21.2: %s", err)
		}

		if err := util.ReplaceInFile("go.mod",
			"k8s.io/component-base v0.24.2",
			"k8s.io/component-base v0.21.2"); err != nil {
			log.Warnf("unable to update the go.mod with k8s.io/component-base v0.21.2: %s", err)
		}

		// Cannot use v1+ unless controller runtime is v0.11
		if err := util.ReplaceInFile("go.mod",
			"github.com/go-logr/logr v1.2.0",
			"github.com/go-logr/logr v0.4.0"); err != nil {
			log.Warnf("unable to update the go.mod with github.com/go-logr/logr v0.4.0: %s", err)
		}

		if err := util.ReplaceInFile("go.mod",
			"github.com/go-logr/zapr v1.2.0",
			"github.com/go-logr/zapr v0.4.0"); err != nil {
			log.Warnf("unable to update the go.mod with github.com/go-logr/zapr v0.4.0: %s", err)
		}

		if err := util.ReplaceInFile("go.mod",
			"k8s.io/klog/v2 v2.60.1",
			"k8s.io/klog/v2 v2.9.0"); err != nil {
			log.Warnf("unable to update the go.mod with k8s.io/klog/v2 v2.9.0: %s", err)
		}

		// Due to some indirect dependency changes with a bump in k8s packages from 0.23.x --> 0.24.x we need to
		// clear out all indirect dependencies before we run `go mod tidy` so that the above changes get resolved correctly
		if err := util.ReplaceRegexInFile("go.mod", `(require \(\n(\t.* \/\/ indirect\n)+\))`, ""); err != nil {
			log.Warnf("unable to update the go.mod indirect dependencies: %s", err)
		}

		err = util.RunCmd("Update dependencies", "go", "mod", "tidy")
		if err != nil {
			return err
		}
	}
	return nil
}
