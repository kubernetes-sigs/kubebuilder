#!/usr/bin/env bash

set -x -e

# Download the dependency binaries
export TOOLS=kubebuilder-tools-1.9-linux-amd64.tar.gz
curl -L https://storage.googleapis.com/kubebuilder-tools/$TOOLS -o /tmp/$TOOLS
mkdir -p /tmp/kubebuilder/bin/
tar xzvf /tmp/$TOOLS -C /tmp/
