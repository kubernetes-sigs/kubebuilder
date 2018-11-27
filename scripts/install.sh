#!/bin/sh

#
# This file will be  fetched as: curl -L https://git.io/getLatestKubebuilder | sh -
# so it should be pure bourne shell, not bash (and not reference other scripts)
#
# The script fetches the latest kubebuilder release candidate and untars it.
# It lets users to do curl -L https://git.io//getLatestKubebuilder | KUBEBUILDER_VERSION=1.0.5 sh -
# for instance to change the version fetched.

# Check if the program is installed, otherwise exit
function command_exists () {
  if ! [ -x "$(command -v $1)" ]; then
    echo "Error: $1 program is not installed." >&2
    exit 1
  fi
}

if [ "x${KUBEBUILDER_VERSION}" = "x" ] ; then
  KUBEBUILDER_VERSION=$(curl -L -s https://api.github.com/repos/kubernetes-sigs/kubebuilder/releases/latest | \
                  grep tag_name | sed "s/ *\"tag_name\": *\"\\(.*\\)\",*/\\1/")
fi

KUBEBUILDER_VERSION=${KUBEBUILDER_VERSION#"v"}
KUBEBUILDER_VERSION_NAME="kubebuilder_${KUBEBUILDER_VERSION}"
KUBEBUILDER_DIR=/usr/local/kubebuilder

# Check if folder containing kubebuilder executable exists and is not empty
if [ -d "$KUBEBUILDER_DIR" ]; then
  if [ "$(ls -A $KUBEBUILDER_DIR)" ]; then
    echo "\n/usr/local/kubebuilder folder is not empty. Please delete or backup it before to install ${KUBEBUILDER_VERSION_NAME}"
    exit 1
  fi
fi

# Check if curl, tar commands/programs exist
command_exists curl
command_exists tar

# Determine OS
OS="$(uname)"
case $OS in
  Darwin)
    OSEXT="darwin"
    ;;
  Linux)
    OSEXT="linux"
    ;;
  *)
    echo "Only OSX and Linux OS are supported !"
    exit 1
    ;;
esac

HW=$(uname -m)
case $HW in
    x86_64)
      ARCH=amd64 ;;
    *)
      echo "Only x86_64 machines are supported !"
      exit 1
      ;;
esac

TMP_DIR=$(mktemp -d)

# Downloading Kuberbuilder compressed file using curl program
URL="https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VERSION}/${KUBEBUILDER_VERSION_NAME}_${OSEXT}_${ARCH}.tar.gz"
echo "Downloading ${KUBEBUILDER_VERSION_NAME}\nfrom $URL\n"
curl -L "$URL"| tar xz -C $TMP_DIR

echo "Downloaded executable files"
ls "$TMP_DIR/${KUBEBUILDER_VERSION_NAME}_${OSEXT}_${ARCH}/bin"

echo "Moving files to $KUBEBUILDER_DIR folder\n"
mv $TMP_DIR/${KUBEBUILDER_VERSION_NAME}_${OSEXT}_${ARCH} $TMP_DIR/kubebuilder && sudo mv -f $TMP_DIR/kubebuilder /usr/local/

rm -rf $TMP_DIR