#!/bin/bash

set -ex

(
    pushd ./utils
    go build -o ../../../bin/literate-go ./literate.go
    popd
) &>/dev/null

../../bin/literate-go "$@"
