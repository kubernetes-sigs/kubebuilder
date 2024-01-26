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

source "$(dirname "$0")/../common.sh"

# This function scaffolds test projects given a project name and flags.
#
# Usage:
#
#   scaffold_test_project <project name> <flag1> <flag2>
function scaffold_test_project {
  local project=$1
  shift
  local init_flags="$@"

  local testdata_dir="$(dirname "$0")/../../testdata"
  mkdir -p $testdata_dir/$project
  rm -rf $testdata_dir/$project/*
  pushd $testdata_dir/$project

  # Remove tool binaries for projects of version 2, which don't have locally-configured binaries,
  # so the correct versions are used. Also, webhooks in version 2 don't have --make flag
  if [[ $init_flags =~ --project-version=2 ]]; then
    rm -f "$(command -v controller-gen)"
    rm -f "$(command -v kustomize)"
  fi

  header_text "Generating project ${project} with flags: ${init_flags}"

  go mod init sigs.k8s.io/kubebuilder/testdata/$project  # our repo autodetection will traverse up to the kb module if we don't do this

  header_text "Initializing project ..."
  $kb init $init_flags --domain testproject.org --license apache2 --owner "The Kubernetes authors"

  if [ $project == "project-v2" ] || [ $project == "project-v3" ] || [ $project == "project-v3-config" ] || [ $project == "project-v4" ] || [ $project == "project-v4-config" ]; then
    header_text 'Creating APIs ...'
    $kb create api --group crew --version v1 --kind Captain --controller=true --resource=true --make=false
    $kb create api --group crew --version v1 --kind Captain --controller=true --resource=true --make=false --force
    $kb create webhook --group crew --version v1 --kind Captain --defaulting --programmatic-validation
    if [ $project == "project-v3" ]; then
      $kb create webhook --group crew --version v1 --kind Captain --defaulting --programmatic-validation --force
    fi

    if [ $project == "project-v2" ]; then
      $kb create api --group crew --version v1 --kind FirstMate --controller=true --resource=true --make=false
    else
      $kb create api --group crew --version v1 --kind FirstMate --controller=true --resource=true --make=false
    fi
    $kb create webhook --group crew --version v1 --kind FirstMate --conversion

    if [ $project == "project-v3" ] || [ $project == "project-v4" ]; then
      $kb create api --group crew --version v1 --kind Admiral --plural=admirales --controller=true --resource=true --namespaced=false --make=false
      $kb create webhook --group crew --version v1 --kind Admiral --plural=admirales --defaulting
    else
      $kb create api --group crew --version v1 --kind Admiral --controller=true --resource=true --namespaced=false --make=false
      $kb create webhook --group crew --version v1 --kind Admiral --defaulting
    fi

    $kb create api --group crew --version v1 --kind Laker --controller=true --resource=false --make=false
  elif [[ $project =~ multigroup ]]; then
    header_text 'Switching to multigroup layout ...'
    $kb edit --multigroup=true

    header_text 'Creating APIs ...'
    $kb create api --group crew --version v1 --kind Captain --controller=true --resource=true --make=false
    $kb create webhook --group crew --version v1 --kind Captain --defaulting --programmatic-validation

    $kb create api --group ship --version v1beta1 --kind Frigate --controller=true --resource=true --make=false
    $kb create webhook --group ship --version v1beta1 --kind Frigate --conversion

    $kb create api --group ship --version v1 --kind Destroyer --controller=true --resource=true --namespaced=false --make=false
    $kb create webhook --group ship --version v1 --kind Destroyer --defaulting

    $kb create api --group ship --version v2alpha1 --kind Cruiser --controller=true --resource=true --namespaced=false --make=false
    $kb create webhook --group ship --version v2alpha1 --kind Cruiser --programmatic-validation

    $kb create api --group sea-creatures --version v1beta1 --kind Kraken --controller=true --resource=true --make=false

    $kb create api --group sea-creatures --version v1beta2 --kind Leviathan --controller=true --resource=true --make=false

    $kb create api --group foo.policy --version v1 --kind HealthCheckPolicy --controller=true --resource=true --make=false

    $kb create api --group apps --version v1 --kind Deployment --controller=true --resource=false --make=false

    $kb create api --group foo --version v1 --kind Bar --controller=true --resource=true --make=false
    $kb create api --group fiz --version v1 --kind Bar --controller=true --resource=true --make=false

    if [ $project == "project-v4-multigroup" ] || [ $project == "project-v4-multigroup-with-deploy-image" ] ; then
      $kb create api --version v1 --kind Lakers --controller=true --resource=true --make=false
      $kb create webhook --version v1 --kind Lakers --defaulting --programmatic-validation
    fi
  elif [[ $project =~ deploy-image ]]; then
      header_text 'Creating Memcached API with deploy-image plugin ...'
      $kb create api --group example.com --version v1alpha1 --kind Memcached --image=memcached:1.4.36-alpine --image-container-command="memcached,-m=64,-o,modern,-v" --image-container-port="11211" --run-as-user="1001" --plugins="deploy-image/v1-alpha" --make=false
      $kb create api --group example.com --version v1alpha1 --kind Busybox --image=busybox:1.28 --plugins="deploy-image/v1-alpha" --make=false
      header_text 'Creating Memcached webhook ...'
      $kb create webhook --group example.com --version v1alpha1 --kind Memcached --programmatic-validation
  fi

  if [[ $project == project-v4-with-grafana ]]; then
      header_text 'Editing project with Grafana plugin ...'
      $kb edit --plugins=grafana.kubebuilder.io/v1-alpha
  fi
  
  make generate manifests
  # TODO fix the error with multigroup layout and allow it be generated
  # with this one.
  # Error: trouble configuring builtin PatchTransformer with config: `
  # path: patches/webhook_in_sea-creatures_krakens.yaml
  # `: failed to get the patch file from path(patches/webhook_in_sea-creatures_krakens.yaml): evalsymlink failure on '/Users/camiladeomacedo/go/src/sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/config/crd/patches/webhook_in_sea-creatures_krakens.yaml' : lstat go/src/sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/config/crd/patches/webhook_in_sea-creatures_krakens.yaml: no such file or directory
  if [[ $project =~ v4 && ! $project =~ multigroup ]]; then
    make build-installer
  fi

  rm -f go.sum
  go mod tidy
  popd
}

build_kb

# [Deprecated] - Project version 2 uses plugin go/v2 (default).
scaffold_test_project project-v2 --project-version=2

# [Deprecated] - Project version 3 (default) uses plugin go/v3 (default).
scaffold_test_project project-v3 --plugins="go/v3"

# [Currently, default CLI plugin] - [Next version, alpha] Project version v4-alpha
scaffold_test_project project-v4 --plugins="go/v4"
scaffold_test_project project-v4-multigroup --plugins="go/v4"
scaffold_test_project project-v4-multigroup-with-deploy-image --plugins="go/v4"
scaffold_test_project project-v4-with-deploy-image --plugins="go/v4"
scaffold_test_project project-v4-with-grafana --plugins="go/v4"
