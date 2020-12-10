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

function usage() {
  echo "
  This script runs goreleaser using the build/.goreleaser.yml config.
  While it can be run locally, it is intended to be run by cloudbuild
  in the goreleaser/goreleaser image.

  GORELEASER_FLAGS: contains flags to pass to the goreleaser binary (default: only --config is set).

  SNAPSHOT: if set to any value, runs goreleaser in snapshot mode with mock release notes (default: unset).

  NOTES_FLAGS: contains flags to pass to the notes binary (sigs.k8s.io/kubebuilder-release-tools/notes).
               Does nothing if SNAPSHOT is set. (default: unset).

  Examples:

  # Run in snapshot mode: fake release notes, nothing is published, binaries build in '$(pwd)/dist'
  \$ SNAPSHOT=1 $0

  # Add a release type to the release notes
  \$ NOTES_FLAGS=\"-r beta\" $0
"
}

# GORELEASER_FLAGS contains flags for goreleaser such that the binary can be run
# in local/snapshot/prod mode from the same script.
# NOTE: if --snapshot is in GORELEASER_FLAGS, the release is not published to GitHub
# and the build is available under $PWD/dist.
GORELEASER_FLAGS="${GORELEASER_FLAGS:-}"
# NOTES_FLAGS contains flags for the release notes generator (see install_notes for details).
NOTES_FLAGS="${NOTES_FLAGS:-}"
# SNAPSHOT is set by the CLI flag parser if --snapshot is a passed flag.
# If not set, release notes are not generated.
SNAPSHOT="${SNAPSHOT:-}"

while [ $# -gt 0 ]; do
  case $1 in
    -h|--help)
    usage
    exit 0
    ;;
  esac
done


# install_notes installs kubebuilder's release notes generator globally with name "notes".
function install_notes() {
  local tmp=$(mktemp -d)
  pushd "$tmp"
  go mod init tmp
  # Get by commit because v0.1.1 cannot be retrieved via `go get`.
  go get sigs.k8s.io/kubebuilder-release-tools/notes@4777888c377a26956f1831d5b9207eea1fa3bf29
  popd
  rm -rf "$tmp"
}

set -o errexit
set -o pipefail

# Generate real release notes.
if [ -z "$SNAPSHOT" ]; then
  tmp_notes="$(mktemp)"
  trap "rm -f ${tmp_notes}" EXIT
  install_notes
  notes $NOTES_FLAGS | tee "$tmp_notes"
  GORELEASER_FLAGS="${GORELEASER_FLAGS} --release-notes=${tmp_notes}"
else
  # TODO(estroz): figure out how to generate snapshot release notes with the kubebuilder generator.
  echo "Running in snapshot mode. Release notes will not be generated from commits."
  notes="Mock Release Notes for $(git describe --tags --always --broken)"
  GORELEASER_FLAGS="${GORELEASER_FLAGS} --snapshot --rm-dist --skip-validate --release-notes <(echo \"${notes}\")"
fi

# eval to run process substitution.
eval goreleaser release --config=build/.goreleaser.yml $GORELEASER_FLAGS
