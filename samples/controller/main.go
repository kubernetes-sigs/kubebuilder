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

package main

import (
	"flag"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"github.com/kubernetes-sigs/kubebuilder/pkg/config"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	"github.com/kubernetes-sigs/kubebuilder/samples/controller/pkg/apis/samplecontroller/v1alpha1"
	"github.com/kubernetes-sigs/kubebuilder/samples/controller/pkg/inject/args"
	"k8s.io/api/apps/v1"
)

func main() {
	// Setup clients, informers, channels for injection
	flag.Parse()
	config := config.GetConfigOrDie()
	rargs := run.RunArguments{Stop: signals.SetupSignalHandler()}
	iargs := args.CreateInjectArgs(config)

	// Start informers
	iargs.ControllerManager.AddInformerProvider(&v1.Deployment{}, iargs.KubernetesInformers.Apps().V1().Deployments())
	iargs.ControllerManager.AddInformerProvider(&v1alpha1.Foo{}, iargs.Informers.Samplecontroller().V1alpha1().Foos())

	// Add the Foo controller
	iargs.ControllerManager.AddController(NewController(iargs))

	// Run the Informers and Controllers
	iargs.ControllerManager.RunInformersAndControllers(rargs)
	<-rargs.Stop
}
