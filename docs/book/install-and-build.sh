#!/bin/bash

set -e

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
echo "downloading mdBook-v0.3.1-${arch}-${target}.${ext}"
set -x
curl -sL -o /tmp/mdbook.${ext} https://github.com/rust-lang-nursery/mdBook/releases/download/v0.3.1/mdBook-v0.3.1-${arch}-${target}.${ext}
${cmd} /tmp/mdbook.${ext}
chmod +x /tmp/mdbook

echo "grabbing the latest released controller-gen"
# TODO(directxman12): remove the @v0.2.0-beta.4 once get v0.2.0 released,
# so that we actually get the latest version
go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.0-beta.4

# make sure we add the go bin directory to our path
gobin=$(go env GOBIN)
gobin=${GOBIN:-$(go env GOPATH)/bin}  # GOBIN won't always be set :-/

export PATH=${gobin}:$PATH
/tmp/mdbook build
