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

    # Default to the first GOPATH if there are multiple ones
    CHECK_GOPATH=$(go env GOPATH)
    if [[ $CHECK_GOPATH == *":"* ]]; then
        GOPATH=$(echo $CHECK_GOPATH | cut -d: -f1)
    fi
    oldgopath=$GOPATH

    if [ $version == "2" ] || [ $version == "3-alpha" ]; then
        header_text "Starting to generate projects with version $version"
        header_text "Generating $project"

        export GO111MODULE=on
        export PATH=$PATH:$GOPATH/bin
        go mod init sigs.k8s.io/kubebuilder/testdata/$project  # our repo autodetection will traverse up to the kb module if we don't do this

        header_text "initializing $project ..."
        $kb init --project-version $version --domain testproject.org --license apache2 --owner "The Kubernetes authors"

        if [ $project == "project-v2" ] || [ $project == "project-v3" ]; then
            header_text 'Creating APIs ...'
            $kb create api --group crew --version v1 --kind Captain --controller=true --resource=true --make=false
            $kb create webhook --group crew --version v1 --kind Captain --defaulting --programmatic-validation
            $kb create api --group crew --version v1 --kind FirstMate --controller=true --resource=true --make=false
            $kb create webhook --group crew --version v1 --kind FirstMate --conversion
            $kb create api --group crew --version v1 --kind Admiral --controller=true --resource=true --namespaced=false --make=false
        elif [ $project == "project-v2-multigroup" ] || [ $project == "project-v3-multigroup" ]; then
            header_text 'Switching to multigroup layout ...'
            $kb edit --multigroup=true

            header_text 'Creating APIs ...'
            $kb create api --group crew --version v1 --kind Captain --controller=true --resource=true --make=false
            $kb create webhook --group crew --version v1 --kind Captain --defaulting --programmatic-validation
            $kb create api --group ship --version v1beta1 --kind Frigate --controller=true --resource=true --make=false
            $kb create webhook --group ship --version v1beta1 --kind Frigate --conversion
            $kb create api --group ship --version v1 --kind Destroyer --controller=true --resource=true --namespaced=false --make=false
            $kb create api --group ship --version v2alpha1 --kind Cruiser --controller=true --resource=true --namespaced=false --make=false
            $kb create api --group sea-creatures --version v1beta1 --kind Kraken --controller=true --resource=true --make=false
            $kb create api --group sea-creatures --version v1beta2 --kind Leviathan --controller=true --resource=true --make=false
            $kb create api --group foo.policy --version v1 --kind HealthCheckPolicy --controller=true --resource=true --make=false
          elif [ $project == "project-v2-addon" ] || [ $project == "project-v3-addon" ]; then
            header_text 'enabling --pattern flag ...'
            export KUBEBUILDER_ENABLE_PLUGINS=1
            header_text 'Creating APIs ...'
            $kb create api --group crew --version v1 --kind Captain --controller=true --resource=true --pattern=addon
            $kb create api --group crew --version v1 --kind FirstMate --controller=true --resource=true --make=false --pattern=addon
            $kb create api --group crew --version v1 --kind Admiral --controller=true --resource=true --namespaced=false --make=false --pattern=addon
        fi
    fi
    make all test
    rm -f go.sum
    rm -rf ./bin
    export GOPATH=$oldgopath
    popd
}

set -e

build_kb
scaffold_test_project project-v2 2
scaffold_test_project project-v2-multigroup 2
scaffold_test_project project-v2-addon 2
scaffold_test_project project-v3 3-alpha
scaffold_test_project project-v3-multigroup 3-alpha
scaffold_test_project project-v3-addon 3-alpha
