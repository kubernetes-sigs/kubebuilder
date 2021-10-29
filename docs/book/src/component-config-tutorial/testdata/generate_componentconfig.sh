#!/usr/bin/env bash

#  Copyright 2020 The Kubernetes Authors
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

#
#  This script is a helper which has just the commands
#  to generate the conjob tutorial to let us know update manually the testdata dir
#  It allows us run ./generate_cronjob.sh and git diff with to check what requires updates
#  NOTE: run make install from the project root before execute it.
#

set -o errexit
set -o pipefail

# Turn colors in this script off by setting the NO_COLOR variable in your
# environment to any value:
#
# $ NO_COLOR=1 test.sh
NO_COLOR=${NO_COLOR:-""}
if [ -z "$NO_COLOR" ]; then
  header=$'\e[1;33m'
  reset=$'\e[0m'
else
  header=''
  reset=''
fi

build_kb() {
    go build -o ./bin/kubebuilder sigs.k8s.io/kubebuilder/cmd
}

function header_text {
  echo "$header$*$reset"
}

function gen_component_config_tutorial {
  header_text "removing project ..."
  rm -rf project
  header_text "starting to generate the component config ..."
  mkdir project
  cd project
  header_text "creating tutorial.kubebuilder.io base  ..."
  kubebuilder init --domain tutorial.kubebuilder.io --repo tutorial.kubebuilder.io/project --component-config --license apache2 --owner "The Kubernetes authors"
  kubebuilder create api --group config --version v2 --kind ProjectConfig --resource --controller=false --make=false
}

gen_component_config_tutorial
