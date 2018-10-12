#!/usr/bin/env bash

#  Copyright 2018 The Kubernetes Authors.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

set -e

go build -o ./bin/kubebuilder sigs.k8s.io/kubebuilder/cmd
rm -rf ./test/project/*
cd test/project
ln -s ../../vendor vendor
../../bin/kubebuilder init --domain testproject.org --license apache2 --owner "The Kubernetes authors" --dep=false
../../bin/kubebuilder create api --group crew --version v1 --kind FirstMate --controller=true --resource=true --make=false
../../bin/kubebuilder alpha webhook --group crew --version v1 --kind FirstMate --type=mutating --operations=create,update
../../bin/kubebuilder alpha webhook --group crew --version v1 --kind FirstMate --type=mutating --operations=delete
../../bin/kubebuilder create api --group ship --version v1beta1 --kind Frigate --example=false --controller=true --resource=true --make=false
../../bin/kubebuilder alpha webhook --group ship --version v1beta1 --kind Frigate --type=validating --operations=update
../../bin/kubebuilder create api --group creatures --version v2alpha1 --kind Kraken --namespaced=false --example=false --controller=true --resource=true --make=false
../../bin/kubebuilder alpha webhook --group creatures --version v2alpha1 --kind Kraken --type=validating --operations=create
../../bin/kubebuilder create api --group core --version v1 --kind Namespace --example=false --controller=true --resource=false --namespaced=false
../../bin/kubebuilder alpha webhook --group core --version v1 --kind Namespace --type=mutating --operations=update
../../bin/kubebuilder create api --group policy --version v1beta1 --kind HealthCheckPolicy --example=false --controller=true --resource=true --namespaced=false
make
rm -rf ./bin/
cd -
