#!/bin/bash

#!/bin/sh
#
# This file will be  fetched as: curl -L https://git.io/getLatestKubebuilder | sh -
# so it should be pure bourne shell, not bash (and not reference other scripts)
#
# The script fetches the latest kubebuilder release candidate and untars it.
# It lets users to do curl -L https://git.io//getLatestKubebuilder | KUBEBUILDER_VERSION=1.0.5 sh -
# for instance to change the version fetched.

OS="$(uname)"
if [ "x${OS}" = "xDarwin" ] ; then
  OSEXT="darwin"
else
  OSEXT="linux"
fi
ARCH=amd64

if [ "x${KUBEBUILDER_VERSION}" = "x" ] ; then
  KUBEBUILDER_VERSION=$(curl -L -s https://api.github.com/repos/kubernetes-sigs/kubebuilder/releases/latest | \
                  grep tag_name | sed "s/ *\"tag_name\": *\"\\(.*\\)\",*/\\1/")
fi

KUBEBUILDER_VERSION=${KUBEBUILDER_VERSION#"v"}
NAME="kubebuilder_${KUBEBUILDER_VERSION}"
URL="https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/${NAME}_${OSEXT}_${ARCH}.tar.gz"
echo "Downloading $NAME from $URL ..."
curl -L "$URL" | tar xz

echo "Downloaded these executable files into $NAME: "
ls "${NAME}_${OSEXT}_${ARCH}/bin"
sudo mv ${NAME}_${OSEXT}_${ARCH} /usr/local/kubebuilder

echo "Add kubebuilder to your path; e.g copy paste in your shell and/or ~/.profile:"
echo "export PATH=\$PATH:/usr/local/kubebuilder/bin"