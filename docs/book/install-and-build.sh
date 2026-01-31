#!/bin/bash

#  Copyright 2020 The Kubernetes Authors.
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

set -e

# The following code is required to allow the preview works with an upper go version
# More info : https://community.netlify.com/t/go-version-1-13/5680
# Get the directory that this script file is in
THIS_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

cd "$THIS_DIR"

if [[ -n "$(command -v gimme)" ]]; then
    GO_VERSION=${GO_VERSION:-stable}  # Use the provided GO_VERSION or default to 'stable'
    eval "$(gimme $GO_VERSION)"
fi
echo go version
GOBIN=$THIS_DIR/functions go install ./...

os=$(go env GOOS)
arch=$(go env GOARCH)

# translate arch to rust's conventions (if we can)
if [[ ${arch} == "amd64" ]]; then
    arch="x86_64"
elif [[ ${arch} == "x86" ]]; then
    arch="i686"
elif [[ ${arch} == "arm64" ]]; then
    # arm64 is not supported for v0.4.40 mdbook, so using x86_64 type.
    # Once the mdbook is upgraded to latest, use 'aarch64'
    arch="x86_64"
fi

# translate os to rust's conventions (if we can)
ext="tar.gz"
cmd="tar -C /tmp -xzvf"
case ${os} in
    windows)
        target="pc-windows-msvc"
        ext="zip"
        cmd="unzip -d /tmp"
        ;;
    darwin)
        target="apple-darwin"
        ;;
    linux)
        # works for linux, too
        target="unknown-${os}-musl"
        ;;
    *)
        target="unknown-${os}"
        ;;
esac

# grab mdbook
# we hardcode linux/amd64 since rust uses a different naming scheme and it's a pain to tran
MDBOOK_VERSION="v0.5.2"
MDBOOK_BASENAME="mdBook-${MDBOOK_VERSION}-${arch}-${target}"
MDBOOK_URL="https://github.com/rust-lang/mdBook/releases/download/${MDBOOK_VERSION}/${MDBOOK_BASENAME}.${ext}"

echo "downloading ${MDBOOK_BASENAME}.${ext} from ${MDBOOK_URL}"
set -x
curl -fL -o /tmp/mdbook.${ext} "${MDBOOK_URL}"
${cmd} /tmp/mdbook.${ext}
chmod +x /tmp/mdbook

CONTROLLER_GEN_VERSION="v0.20.0"

echo "grabbing the controller-gen version: ${CONTROLLER_GEN_VERSION}"
go version
go install sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION}

# make sure we add the go bin directory to our path
gobin=$(go env GOBIN)
gobin=${gobin:-$(go env GOPATH)/bin} # GOBIN won't always be set :-/

export PATH=${gobin}:$PATH
verb=${1:-build}
/tmp/mdbook ${verb}
