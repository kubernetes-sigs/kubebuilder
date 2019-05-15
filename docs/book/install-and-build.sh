#!/bin/bash

os=$(go env GOOS)
arch=$(go env GOARCH)

# grab mdbook
# mdbook's deploy got borked by a CI move, so grab our build till that gets
# fixed (https://github.com/rust-lang-nursery/mdBook/issues/904).
curl -sL -o /tmp/mdbook https://storage.googleapis.com/kubebuilder-build-tools/mdbook-0.2.3-${os}-${arch}
chmod +x /tmp/mdbook
/tmp/mdbook build
