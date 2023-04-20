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

#  This script is a helper which has just the commands
#  to generate the multiversion tutorial to let us know update manually the testdata dir
#  It allows us run ./generate_mutiversion.sh and git diff with to check what requires updates
#  NOTE: run make install from the project root before execute it.

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

function gen_cronjob_tutorial {
  header_text "removing project ..."
  rm -rf project
  header_text "starting to generate the cronjob ..."
  mkdir project
  cd project
  header_text "generate base  ..."
  kubebuilder init --plugins="go/v4" --domain=tutorial.kubebuilder.io --project-version=3 --repo=tutorial.kubebuilder.io/project --license apache2 --owner "The Kubernetes authors"
  kubebuilder create api --group batch --version v1 --kind CronJob --resource --controller --make=false
  kubebuilder create webhook --group batch --version v1 --kind CronJob --defaulting --programmatic-validation
  kubebuilder create webhook --group batch --version v1 --kind CronJob --conversion --force
  kubebuilder create api --group batch --version v2 --kind CronJob --resource=true --controller=false
  kubebuilder create webhook --group batch --version v2 --kind CronJob --defaulting --programmatic-validation
  kubebuilder create webhook --group batch --version v2 --kind CronJob --conversion --force
  go mod tidy
  make
}


gen_cronjob_tutorial
