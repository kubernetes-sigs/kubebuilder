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

clear
. $(dirname ${BASH_SOURCE})/util.sh

desc "Initialize Go modules"
run "go mod init demo.kubebuilder.io"

desc "Let's initialize the project"
run "kubebuilder init --domain tutorial.kubebuilder.io"
clear

desc "Examine scaffolded files..."
run "tree ."
clear

desc "Create our custom cronjob api"
run "kubebuilder create api --group batch --version v1 --kind CronJob"
clear

desc "Let's take a look at the API and Controller files"
run "tree ./api ./controllers"
clear

desc "Install CRDs in Kubernetes cluster"
run "make install"
clear

desc "Run controller manager locally"
run "make run"
