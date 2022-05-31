#!/bin/bash
KUSTOMIZE_VERSION=$1
INSTALL_PATH=$2
if [ ! -f $INSTALL_PATH/kustomize ] 
then

OS=$(uname -s | awk '{ print tolower($0) }')
ARCH=$(uname -m)

echo "Building kustomize $KUSTOMIZE_VERSION locally for $OS/$ARCH"

TEMP_DIR=$(mktemp -d)
cd $TEMP_DIR
git clone --depth 1 --branch kustomize/$KUSTOMIZE_VERSION https://github.com/kubernetes-sigs/kustomize.git
cd kustomize/kustomize
GOBIN=$INSTALL_PATH go install .
cd ../../..
rm -rf $TEMP_DIR
fi