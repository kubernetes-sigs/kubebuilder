#!/bin/bash

set -ex

(
    pushd ./utils
    go build -o ../../../bin/literate-go ./litgo
    popd
) &>/dev/null

../../bin/literate-go "$@"
