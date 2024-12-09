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

  header_text "Generating project ${project} with flags: ${init_flags}"
  go mod init sigs.k8s.io/kubebuilder/testdata/$project  # our repo autodetection will traverse up to the kb module if we don't do this
  header_text "Initializing project ..."
  $kb init $init_flags --domain testproject.org --license apache2 --owner "The Kubernetes authors"

  if [ $project == "project-v4" ] ; then
    header_text 'Creating APIs ...'
    $kb create api --group crew --version v1 --kind Captain --controller=true --resource=true --make=false
    $kb create api --group crew --version v1 --kind Captain --controller=true --resource=true --make=false --force
    $kb create webhook --group crew --version v1 --kind Captain --defaulting --programmatic-validation --make=false

    # Create API to test conversion from v1 to v2
    $kb create api --group crew --version v1 --kind FirstMate --controller=true --resource=true --make=false
    $kb create api --group crew --version v2 --kind FirstMate --controller=false --resource=true --make=false
    $kb create webhook --group crew --version v1 --kind FirstMate --conversion --make=false --spoke v2

    $kb create api --group crew --version v1 --kind Admiral --plural=admirales --controller=true --resource=true --namespaced=false --make=false
    $kb create webhook --group crew --version v1 --kind Admiral --plural=admirales --defaulting
    # Controller for External types
    $kb create api --group "cert-manager" --version v1 --kind Certificate --controller=true --resource=false --make=false --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 --external-api-domain=io
    # Webhook for External types
    $kb create webhook --group "cert-manager" --version v1 --kind Issuer --defaulting --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 --external-api-domain=io
    # Webhook for Core type
    $kb create webhook --group core --version v1 --kind Pod --defaulting
    # Webhook for kubernetes Core type that is part of an api group
    $kb create webhook --group apps --version v1 --kind Deployment --defaulting --programmatic-validation
  fi

  if [[ $project =~ multigroup ]]; then
    header_text 'Switching to multigroup layout ...'
    $kb edit --multigroup=true

    header_text 'Creating APIs ...'
    $kb create api --group crew --version v1 --kind Captain --controller=true --resource=true --make=false
    $kb create webhook --group crew --version v1 --kind Captain --defaulting --programmatic-validation --make=false

    $kb create api --group ship --version v1beta1 --kind Frigate --controller=true --resource=true --make=false
    $kb create api --group ship --version v1 --kind Destroyer --controller=true --resource=true --namespaced=false --make=false
    $kb create webhook --group ship --version v1 --kind Destroyer --defaulting --make=false
    $kb create api --group ship --version v2alpha1 --kind Cruiser --controller=true --resource=true --namespaced=false --make=false
    $kb create webhook --group ship --version v2alpha1 --kind Cruiser --programmatic-validation --make=false

    $kb create api --group sea-creatures --version v1beta1 --kind Kraken --controller=true --resource=true --make=false
    $kb create api --group sea-creatures --version v1beta2 --kind Leviathan --controller=true --resource=true --make=false
    $kb create api --group foo.policy --version v1 --kind HealthCheckPolicy --controller=true --resource=true --make=false
    # Controller for Core types
    $kb create api --group apps --version v1 --kind Deployment --controller=true --resource=false --make=false
    $kb create api --group foo --version v1 --kind Bar --controller=true --resource=true --make=false
    $kb create api --group fiz --version v1 --kind Bar --controller=true --resource=true --make=false
    # Controller for External types
    $kb create api --group "cert-manager" --version v1 --kind Certificate --controller=true --resource=false --make=false --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 --external-api-domain=io
    # Webhook for External types
    $kb create webhook --group "cert-manager" --version v1 --kind Issuer --defaulting --external-api-path=github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1 --external-api-domain=io
    # Webhook for Core type
    $kb create webhook --group core --version v1 --kind Pod --programmatic-validation --make=false
    # Webhook for kubernetes Core type that is part of an api group
    $kb create webhook --group apps --version v1 --kind Deployment --defaulting --programmatic-validation --make=false
  fi

  if [[ $project =~ multigroup ]] || [[ $project =~ with-plugins ]] ; then
    header_text 'With Optional Plugins ...'
    header_text 'Creating APIs with deploy-image plugin ...'
    $kb create api --group example.com --version v1alpha1 --kind Memcached --image=memcached:1.6.26-alpine3.19 --image-container-command="memcached,--memory-limit=64,-o,modern,-v" --image-container-port="11211" --run-as-user="1001" --plugins="deploy-image/v1-alpha" --make=false
    $kb create api --group example.com --version v1alpha1 --kind Busybox --image=busybox:1.36.1 --plugins="deploy-image/v1-alpha" --make=false
    # Create only validation webhook for Memcached
    $kb create webhook --group example.com --version v1alpha1 --kind Memcached --programmatic-validation --make=false
    # Create API to check webhook --conversion from v1 to v2
    $kb create api --group example.com --version v1 --kind Wordpress --controller=true --resource=true  --make=false
    $kb create api --group example.com --version v2 --kind Wordpress --controller=false --resource=true  --make=false
    $kb create webhook --group example.com --version v1 --kind Wordpress --conversion --make=false --spoke v2

    header_text 'Editing project with Grafana plugin ...'
    $kb edit --plugins=grafana.kubebuilder.io/v1-alpha
  fi

  make all
  make build-installer

  if [[ $project =~ with-plugins ]] ; then
    header_text 'Editing project with Helm plugin ...'
    $kb edit --plugins=helm.kubebuilder.io/v1-alpha
  fi

  # To avoid conflicts
  rm -f go.sum
  go mod tidy
  popd
}

build_kb

scaffold_test_project project-v4 --plugins="go/v4"
scaffold_test_project project-v4-multigroup --plugins="go/v4"
scaffold_test_project project-v4-with-plugins --plugins="go/v4"
