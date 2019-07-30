#!/bin/bash

set -ex

(
    pushd ./utils
    go build -o ../../../bin/marker-docs ./markerdocs
    popd
) &>/dev/null

../../bin/marker-docs "$@"
