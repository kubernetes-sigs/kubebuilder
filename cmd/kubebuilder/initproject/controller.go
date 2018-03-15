/*
Copyright 2017 The Kubernetes Authors.

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

package initproject

import (
	"fmt"
	"path/filepath"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
)

func createControllerManager(boilerplate string) {
	fmt.Printf("\t%s/\n", filepath.Join("cmd", "controller-manager"))
	execute(
		filepath.Join("cmd", "controller-manager", "main.go"),
		"main-template",
		controllerManagerTemplate,
		controllerManagerTemplateArguments{boilerplate, util.Repo},
	)
}

type controllerManagerTemplateArguments struct {
	BoilerPlate string
	Repo        string
}

var controllerManagerTemplate = `{{.BoilerPlate}}

package main

import (
	"flag"
	"log"

    // Import auth/gcp to connect to GKE clusters remotely
    _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

    configlib "github.com/kubernetes-sigs/kubebuilder/pkg/config"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	"github.com/kubernetes-sigs/kubebuilder/pkg/install"
    "github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

    "{{.Repo}}/pkg/inject"
    "{{.Repo}}/pkg/inject/args"
)

var installCRDs = flag.Bool("install-crds", true, "install the CRDs used by the controller as part of startup")

// Controller-manager main.
func main() {
	flag.Parse()

    stopCh := signals.SetupSignalHandler()
	
    config := configlib.GetConfigOrDie()

    if *installCRDs {
        if err := install.NewInstaller(config).Install(&InstallStrategy{crds: inject.Injector.CRDs}); err != nil {
            log.Fatalf("Could not create CRDs: %v", err)
        }
    }

    // Start the controllers
    if err := inject.RunAll(run.RunArguments{Stop: stopCh}, args.CreateInjectArgs(config)); err != nil {
        log.Fatalf("%v", err)
    }
}

type InstallStrategy struct {
	install.EmptyInstallStrategy
	crds []*extensionsv1beta1.CustomResourceDefinition
}

func (s *InstallStrategy) GetCRDs() []*extensionsv1beta1.CustomResourceDefinition {
	return s.crds
}
`
