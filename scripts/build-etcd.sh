#!/bin/bash

OS=$(uname -s | awk '{ print tolower($0) }')
ARCH=$(uname -m)
echo "Building etcd locally for $OS/$ARCH"

TEMP_DIR=$(mktemp -d)
cd $TEMP_DIR
git clone https://github.com/etcd-io/etcd.git
cd etcd
make build
if [ -d $HOME/bin ]
then
    mv ./bin/etcd $HOME/bin/etcd
else
    mkdir $HOME/bin
    mv ./bin/etcd $HOME/bin/etcd
fi
cd ../../
rm -rf $TEMP_DIR
