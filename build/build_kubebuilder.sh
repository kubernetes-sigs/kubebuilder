#!/usr/bin/env bash

# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script runs goreleaser using the build/.goreleaser.yml config.
# While it can be run locally, it is intended to be run by cloudbuild
# in the goreleaser/goreleaser image.

# echo_run echos then evaluates all args.
function echo_run() {
  echo $@
  # Ensure process substitution is evaluated with eval.
  eval $@
}

# install_notes installs kubebuilder's release notes generator globally with name "notes".
function install_notes() {
  local tmp=$(mktemp -d)
  pushd "$tmp"
  go mod init tmp
  go get sigs.k8s.io/kubebuilder-release-tools/notes
  popd
  rm -rf "$tmp"
}

set -o errexit
set -o pipefail

# SNAPSHOT is set by the CLI flag parser if --snapshot is a passed flag.
# If not set, release notes are not generated.
SNAPSHOT=
# GORELEASER_FLAGS sets up goreleaser flags such that it can be run
# in local/snapshot/prod mode from the same script.
# NOTE: if --snapshot is set, release is not published to GitHub
# and the build is available under $PWD/dist.
GORELEASER_FLAGS=

while [[ $# -gt 0 ]]; do
  key="$1"
  case $key in
    --snapshot)
    # TODO(estroz): figure out how to generate snapshot release notes with the kubebuilder generator.
    echo "Running in snapshot mode. Release notes will not be generated from commits."
    notes="Mock Release Notes for $(git describe --tags --always --broken)"
    GORELEASER_FLAGS="${key} --release-notes <(echo \"${notes}\")"
    SNAPSHOT=1
    shift
    ;;
    *)
    GORELEASER_FLAGS="$GORELEASER_FLAGS ${key}"
    shift
    ;;
  esac
done

# Generate real release notes.
if [ -z "$SNAPSHOT" ]; then
  tmp_notes="$(mktemp)"
  trap "rm -f ${tmp_notes}" EXIT
  install_notes
  notes | tee "$tmp_notes"
  GORELEASER_FLAGS="${GORELEASER_FLAGS} --release-notes=${tmp_notes}"
fi

echo_run goreleaser release --config=build/.goreleaser.yml --rm-dist --skip-validate $GORELEASER_FLAGS
