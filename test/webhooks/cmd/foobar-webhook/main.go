/*
Copyright 2018 The Kubernetes Authors.

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

	"github.com/kubernetes-sigs/kubebuilder/pkg/webhooks/"

	"github.com/golang/glog"

	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	foobarwebhook "github.com/kubernetes-sigs/kubebuilder/test/webhooks/webhook"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	flag.Parse()
	glog.Info("starting the FooBar Webhook...")

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	clusterConfig, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatal(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		glog.Fatal(err)
	}

	options := webhook.ControllerOptions{
		ServiceName:      "foobar-webhook",
		ServiceNamespace: "foobar-system",
		Port:             443,
		SecretName:       "foobar-webhook-certs",
		WebhookName:      "webhook.foobar.com",
	}
	controller, err := foobarwebhook.NewAdmissionController(clientset, options)
	if err != nil {
		glog.Fatal(err)
	}
	controller.Run(stopCh)
}
