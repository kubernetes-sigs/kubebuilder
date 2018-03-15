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

package resource

import (
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
	"path/filepath"
)

func doSharedInformers(dir string, args resourceTemplateArgs) bool {
	path := filepath.Join(dir, "pkg", "controller", "sharedinformers", "informers.go")
	return util.WriteIfNotFound(path, "sharedinformer-template", sharedInformersTemplate, args)
}

var sharedInformersTemplate = `
{{.BoilerPlate}}

package sharedinformers

// EDIT THIS FILE!
// Created by "kubebuilder create resource" for you to start additional shared informers

// SetupKubernetesTypes registers the config for watching Kubernetes types
func (si *SharedInformers) SetupKubernetesTypes() bool {
    return true
}

// StartAdditionalInformers starts watching Deployments
func (si *SharedInformers) StartAdditionalInformers(shutdown <-chan struct{}) {
    // Start specific Kubernetes API informers here.  Note, it is only necessary
    // to start 1 informer for each Kind. (e.g. only 1 Deployment informer)

    // Uncomment this to start listening for Deployment Create / Update / Deletes
    // go si.KubernetesFactory.Apps().V1beta1().Deployments().Informer().RunInformersAndControllers(shutdown)

    // INSERT YOUR CODE HERE - start additional shared informers
}
`
