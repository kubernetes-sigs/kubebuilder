#!/bin/bash
# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Set up working directory in /tmp for clean demo
cd /tmp
rm -rf kubebuilder-demo-project
mkdir kubebuilder-demo-project
cd kubebuilder-demo-project

clear
. $(dirname ${BASH_SOURCE})/util.sh

desc "Check if Kubebuilder is installed"
run "kubebuilder version"

desc "Initialize Go modules for our project"
run "go mod init demo.kubebuilder.io/webapp-operator"

desc "Initialize a new Kubebuilder project with custom domain"
run "kubebuilder init --domain demo.kubebuilder.io --repo demo.kubebuilder.io/webapp-operator"
clear

desc "Examine the scaffolded project structure"
run "tree -L 2 ."
clear

desc "Create our first API - a Guestbook with validation markers"
run "kubebuilder create api --group webapp --version v1 --kind Guestbook"
clear

desc "Let's explore the generated files structure"
run "tree api/ internal/controller/"
clear

desc "Look at the API definition - notice the validation markers"
run "cat api/v1/guestbook_types.go"
clear

desc "Now let's create a second API using the modern deploy-image plugin"
run "kubebuilder create api --group webapp --version v1alpha1 --kind Busybox --image=busybox:1.36.1 --plugins=deploy-image/v1-alpha"
clear

desc "Generate manifests including CRDs with OpenAPI validation schemas"
run "make manifests"

desc "Examine the generated CRD with validation rules"
run "head -60 config/crd/bases/webapp.demo.kubebuilder.io_guestbooks.yaml"
clear

desc "Install our Custom Resource Definitions into the cluster"
run "make install"

desc "Verify our CRDs are installed"
run "kubectl get crd --context kind-kubebuilder-demo"
clear

desc "Look at the sample resources that were generated"
run "ls config/samples/"

desc "Show a sample resource"
run "cat config/samples/webapp_v1_guestbook.yaml"
clear

desc "Apply the sample resources to test our APIs"
run "kubectl apply -k config/samples/webapp_v1alpha1_busybox.yaml --context kind-kubebuilder-demo"

desc "Check the created custom resources in the cluster"
run "kubectl get guestbooks,busyboxes -A --context kind-kubebuilder-demo"
clear

desc "Demo completed! The controller can be run with 'make run'"
run "echo 'To run the controller: make run'"
