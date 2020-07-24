#!/bin/sh

#  Copyright 2020 The Kubernetes Authors.
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

# This file will be  fetched as: curl -L https://git.io/getLatestKubebuilder | sh -
# so it should be pure bourne shell, not bash (and not reference other scripts)

set -eu

# To use envtest is required to have etcd, kube-apiserver and kubetcl binaries installed locally.
# This script will create the directory testbin and perform this setup for linux or mac os x envs in

# Kubernetes version e.g v1.18.2
K8S_VER=$1
# ETCD version e.g v3.4.3
ETCD_VER=$2
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/')
ETCD_EXT="tar.gz"
TESTBIN_DIR=testbin

setup_testenv_bin() {
  # Do nothing if the $TESTBIN_DIR directory exist already.
  if [ ! -d $TESTBIN_DIR ]; then
    mkdir -p $TESTBIN_DIR

    # install etcd binary
    # the extension for linux env is not equals for mac os x
    if [ $OS == "darwin" ]; then
       ETCD_EXT="zip"
    fi
    [[ -x ${TESTBIN_DIR}/etcd ]] || curl -L https://storage.googleapis.com/etcd/${ETCD_VER}/etcd-${ETCD_VER}-${OS}-${ARCH}.${ETCD_EXT} | tar zx -C ${TESTBIN_DIR} --strip-components=1 etcd-${ETCD_VER}-${OS}-${ARCH}/etcd

    # install kube-apiserver and kubetcl binaries
    if [ $OS == "darwin" ]
    then
      # kubernetes do not provide the kubernetes-server for darwin,
      # In this way, to have the kube-apiserver is required to build it locally
      # if the project is cloned locally already do nothing
      if [ ! -d $GOPATH/src/k8s.io/kubernetes ]; then
      git clone https://github.com/kubernetes/kubernetes $GOPATH/src/k8s.io/kubernetes --depth=1 -b ${K8S_VER}
      fi

      # if the kube-apiserve is built already then, just copy it
      if [ ! -f $GOPATH/src/k8s.io/kubernetes/_output/local/bin/darwin/amd64/kube-apiserver ]; then
      DIR=$(pwd)
      cd $GOPATH/src/k8s.io/kubernetes
      # Build for linux first otherwise it won't work for darwin - :(
      export KUBE_BUILD_PLATFORMS="linux/amd64"
      make WHAT=cmd/kube-apiserver
      export KUBE_BUILD_PLATFORMS="darwin/amd64"
      make WHAT=cmd/kube-apiserver
      cd ${DIR}
      fi
      cp $GOPATH/src/k8s.io/kubernetes/_output/local/bin/darwin/amd64/kube-apiserver $TESTBIN_DIR/

      # setup kubectl binary
      curl -LO https://storage.googleapis.com/kubernetes-release/release/${K8S_VER}/bin/darwin/amd64/kubectl
      chmod +x kubectl
      mv kubectl $TESTBIN_DIR/

      # allow run the tests without the Mac OS Firewall popup shows for each execution
      codesign --deep --force --verbose --sign - ./${TESTBIN_DIR}/kube-apiserver
    else
      [[ -x $TESTBIN_DIR/kube-apiserver && -x ${TESTBIN_DIR}/kubectl ]] || curl -L https://dl.k8s.io/${K8S_VER}/kubernetes-server-${OS}-${ARCH}.tar.gz | tar zx -C ${TESTBIN_DIR} --strip-components=3 kubernetes/server/bin/kube-apiserver kubernetes/server/bin/kubectl
    fi
  fi
  export PATH=/$TESTBIN_DIR:$PATH
  export TEST_ASSET_KUBECTL=/$TESTBIN_DIR/kubectl
  export TEST_ASSET_KUBE_APISERVER=/$TESTBIN_DIR/kube-apiserver
  export TEST_ASSET_ETCD=/$TESTBIN_DIR/etcd
}

setup_testenv_bin
