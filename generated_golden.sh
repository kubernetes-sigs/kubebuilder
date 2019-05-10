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

source common.sh

build_kb() {
	go build -o ./bin/kubebuilder sigs.k8s.io/kubebuilder/cmd
}


#
# This function scaffolds test projects given a project name and
# project-version.
#
scaffold_test_project() {
	project=$1
	version=$2
    testdata_dir=$(pwd)/testdata
	mkdir -p ./testdata/$project
	rm -rf ./testdata/$project/*
	pushd . 
	cd testdata/$project

	kb=$testdata_dir/../bin/kubebuilder

    oldgopath=$GOPATH
	if [ $version == "1" ]; then
        export GO111MODULE=auto
        export GOPATH=$(pwd)/../.. # go ignores vendor under testdata, so fake out a gopath
        # untar Gopkg.lock and vendor directory for appropriate project version
        tar -zxf $testdata_dir/vendor.v$version.tgz

        $kb init --project-version $version --domain testproject.org --license apache2 --owner "The Kubernetes authors" --dep=false
		$kb create api --group crew --version v1 --kind FirstMate --controller=true --resource=true --make=false
		$kb alpha webhook --group crew --version v1 --kind FirstMate --type=mutating --operations=create,update --make=false
		$kb alpha webhook --group crew --version v1 --kind FirstMate --type=mutating --operations=delete --make=false
		$kb create api --group ship --version v1beta1 --kind Frigate --example=false --controller=true --resource=true --make=false
		$kb alpha webhook --group ship --version v1beta1 --kind Frigate --type=validating --operations=update --make=false
		$kb create api --group creatures --version v2alpha1 --kind Kraken --namespaced=false --example=false --controller=true --resource=true --make=false
		$kb alpha webhook --group creatures --version v2alpha1 --kind Kraken --type=validating --operations=create --make=false
		$kb create api --group core --version v1 --kind Namespace --example=false --controller=true --resource=false --namespaced=false --make=false
		$kb alpha webhook --group core --version v1 --kind Namespace --type=mutating --operations=update --make=false
		$kb create api --group policy --version v1beta1 --kind HealthCheckPolicy --example=false --controller=true --resource=true --namespaced=false --make=false
	elif [ $version == "2" ]; then
        export GO111MODULE=on
        export PATH=$PATH:$(go env GOPATH)/bin
        go mod init sigs.k8s.io/kubebuilder/testdata/project_v2  # our repo autodetection will traverse up to the kb module if we don't do this

        $kb init --project-version $version --domain testproject.org --license apache2 --owner "The Kubernetes authors"
		$kb create api --group crew --version v1 --kind Captain --controller=true --resource=true --make=false
		$kb create api --group crew --version v1 --kind FirstMate --controller=true --resource=true --make=false
		$kb alpha webhook --group crew --version v1 --kind FirstMate --type=mutating --operations=create,update --make=false
		$kb alpha webhook --group crew --version v1 --kind FirstMate --type=mutating --operations=delete --make=false
		# TODO(droot): Adding a second group is a valid test case and kubebuilder is expected to report an error in this case. It
		# doesn't do that currently so leaving it commented so that we can enable it later.
		# $kb create api --group ship --version v1beta1 --kind Frigate --example=false --controller=true --resource=true --make=false
		$kb create api --group core --version v1 --kind Namespace --example=false --controller=true --resource=false --namespaced=false --make=false
		$kb alpha webhook --group core --version v1 --kind Namespace --type=mutating --operations=update --make=false
		# $kb create api --group policy --version v1beta1 --kind HealthCheckPolicy --example=false --controller=true --resource=true --namespaced=false --make=false
	fi
	make all test # v2 doesn't test by default
	rm -f Gopkg.lock
	rm -rf ./vendor
	rm -rf ./bin
    export GOPATH=$oldgopath
	popd
}

set -e
build_kb
scaffold_test_project gopath/src/project 1
scaffold_test_project project_v2 2
