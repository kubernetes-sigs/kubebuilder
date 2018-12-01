#!/bin/sh

source ./scripts/install.sh

echo "Start test"

if [ -z "$KUBEBUILDER_VERSION" ]; then
    echo "\nUnable to fetch the latest version tag. Quit the test"
    exit 0
fi

echo "1. Check whether $TMP_DIR folder is removed.\n"
if [ -d "$TMP_DIR" ]; then
  echo "$TMP_DIR folder is not removed."
  exit 1
fi
echo "passed\n"

echo "2. Check whether $KUBEBUILDER_DIR folder exists.\n"
if [ ! -d "$KUBEBUILDER_DIR" ]; then
  echo "$KUBEBUILDER_DIR folder is not existed."
  exit 1
fi
echo "passed\n"

echo "3. Check whether kubebuilder is installed properly.\n"
if [ ! -x "$KUBEBUILDER_DIR/bin/kubebuilder" ]; then
  echo "$KUBEBUILDER_DIR/bin/kubebuilder is not existed or execute permission not granted."
  exit 1
fi
echo "passed\n"

echo "4. Check whether kubebuilder version is same as installed.\n"
KUBEBUILDER_VERSION_FROM_BIN=$($KUBEBUILDER_DIR/bin/kubebuilder version | \
      sed 's/^.*KubeBuilderVersion:"\(.*\)", KubernetesVendor.*$/\1/')
if [ ! "${KUBEBUILDER_VERSION_FROM_BIN}" = "${KUBEBUILDER_VERSION}" ]; then
  echo "kubebuilder version ${KUBEBUILDER_VERSION_FROM_BIN} mismatched from the version installed (${KUBEBUILDER_VERSION})"
  exit 1
fi
echo "passed\n"

echo "install test done successfully\n"
exit 0
