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

[[ -n "$(command -v gimme)" ]] && eval "$(gimme stable)"
echo go version
GOBIN=$THIS_DIR/functions go install ./...

os=$(go env GOOS)
arch=$(go env GOARCH)

# translate arch to rust's conventions (if we can)
if [[ ${arch} == "amd64" ]]; then
    arch="x86_64"
elif [[ ${arch} == "x86" ]]; then
    arch="i686"
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
        target="unknown-${os}-gnu"
        ;;
    *)
        target="unknown-${os}"
        ;;
esac

# grab mdbook
# we hardcode linux/amd64 since rust uses a different naming scheme and it's a pain to tran
if [ ! -f /tmp/mdbook ]; then
    echo "downloading mdBook-v0.3.1-${arch}-${target}.${ext}"
    set -x
    curl -sL -o /tmp/mdbook.${ext} https://github.com/rust-lang-nursery/mdBook/releases/download/v0.3.1/mdBook-v0.3.1-${arch}-${target}.${ext}
    ${cmd} /tmp/mdbook.${ext}
    chmod +x /tmp/mdbook
fi

echo "grabbing the latest released controller-gen"
go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0

# make sure we add the go bin directory to our path
gobin=$(go env GOBIN)
gobin=${GOBIN:-$(go env GOPATH)/bin}  # GOBIN won't always be set :-/

export PATH=${gobin}:$PATH
verb=${1:-build}
/tmp/mdbook ${verb}
