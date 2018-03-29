/*
Copyright 2016 The Kubernetes Authors.

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

package test

import (
	"time"

	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/client-go/rest"

	"github.com/kubernetes-sig-testing/frameworks/integration"
	"github.com/kubernetes-sigs/kubebuilder/pkg/install"
)

// TestEnvironment creates a Kubernetes test environment that will start / stop the Kubernetes control plane and
// install extension APIs
type TestEnvironment struct {
	ControlPlane integration.ControlPlane
	Config       *rest.Config
	CRDs         []*extensionsv1beta1.CustomResourceDefinition
}

// Stop stops a running server
func (te *TestEnvironment) Stop() {
	te.ControlPlane.Stop()
}

// Start starts a local Kubernetes server and updates te.ApiserverPort with the port it is listening on
func (te *TestEnvironment) Start() (*rest.Config, error) {
	// Start the control plane - retry if it fails
	var err error
	for i := 0; i < 5; i++ {
		err = te.ControlPlane.Start()
		if err == nil {
			break
		}
		te.ControlPlane = integration.ControlPlane{}
	}
	// Give up trying to start the control plane
	if err != nil {
		return nil, err
	}

	// Create the *rest.Config for creating new clients
	te.Config = &rest.Config{
		Host: te.ControlPlane.APIURL().Host,
	}

	// Add CRDs to the apiserver
	err = install.NewInstaller(te.Config).Install(&InstallStrategy{crds: te.CRDs})

	// Wait for discovery service to register CRDs
	// TODO: Poll for this or find a better way of ensuring CRDs are registered in discovery
	time.Sleep(time.Second * 1)

	return te.Config, err
}

type InstallStrategy struct {
	install.EmptyInstallStrategy
	crds []*extensionsv1beta1.CustomResourceDefinition
}

func (s *InstallStrategy) GetCRDs() []*extensionsv1beta1.CustomResourceDefinition {
	return s.crds
}
