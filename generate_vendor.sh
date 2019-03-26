#!/usr/bin/env bash

#  Copyright 2019 The Kubernetes Authors.
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
# Script to generate vendor archive that contains vendor and Gopkg.toml for a
# given project version
#
set -e

go_workspace=''
for p in ${GOPATH//:/ }; do
  if [[ $PWD/ = $p/* ]]; then
    go_workspace=$p
  fi
done

if [ -z $go_workspace ]; then
  echo 'Current directory is not in $GOPATH' >&2
  exit 1
fi

build_kb() {
	rm -f /tmp/kb && \
	go build -o /tmp/kb sigs.k8s.io/kubebuilder/cmd
}


#
# generate_vendor takes project version as input and creates vendor archive
# containing Go dependencies for a Kubebuilder project along with the Gopkg.lock
# file.
#
generate_vendor() {
	version=$1
	project_dir=${go_workspace}/src/sigs.k8s.io/kubebuilder-test
	mkdir -p ${project_dir}
	rm -rf ${project_dir}/*
	pushd . 
	cd ${project_dir}
	/tmp/kb init --project-version $version --domain testproject.org --license apache2 --owner "The Kubernetes authors" --dep=true
	make
	tar -zcvf vendor.v$version.tgz vendor Gopkg.lock && \
	echo "vendor archieve vendor.v$version.tgz is ready."
	popd
}

build_kb && \
generate_vendor $1
