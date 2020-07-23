set -eu

# */
# To use envtest is required etcd, kube-apiserver and kubetcl binaries in the testbin directory.
# This script will perform this setup for linux or mac os x envs.
# */

K8S_VER=v1.18.2
ETCD_VER=v3.4.3
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/')
ETCD_EXT="tar.gz"
TESTBIN_DIR=testbin

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
	  git clone https://github.com/kubernetes/kubernetes $GOPATH/src/k8s.io/kubernetes --depth=1 -b v1.18.2
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
    [[ -x testbin/kube-apiserver && -x ${TESTBIN_DIR}/kubectl ]] || curl -L https://dl.k8s.io/${K8S_VER}/kubernetes-server-${OS}-${ARCH}.tar.gz | tar zx -C ${TESTBIN_DIR} --strip-components=3 kubernetes/server/bin/kube-apiserver kubernetes/server/bin/kubectl
  fi
fi
