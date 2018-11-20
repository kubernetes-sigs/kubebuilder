#!/bin/bash

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
KUBEBUILDER_VERSION_NAME="kubebuilder_${KUBEBUILDER_VERSION}"
URL="https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/${KUBEBUILDER_VERSION_NAME}_${OSEXT}_${ARCH}.tar.gz"
echo "Downloading ${KUBEBUILDER_VERSION_NAME} from $URL ..."
curl -L "$URL" | tar xz

echo "Downloaded these executable files into $NAME: "
ls "${KUBEBUILDER_VERSION_NAME}_${OSEXT}_${ARCH}/bin"
mv ${KUBEBUILDER_VERSION_NAME}_${OSEXT}_${ARCH} kubebuilder && sudo mv -f kubebuilder /usr/local/
RETVAL=$?

if [ $RETVAL -eq 0 ]; then
  echo "Add kubebuilder to your path; e.g copy paste in your shell and/or ~/.profile:"
  echo "export PATH=\$PATH:/usr/local/kubebuilder/bin"
else
  echo "\n/usr/local/kubebuilder folder is not empty. Please delete or backup it before to install ${KUBEBUILDER_VERSION_NAME}"
fi