#  Copyright 2021 The Kubernetes Authors.
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

#!/usr/bin/env bash

# This script effectively retags the quay.io/brancz/kube-rbac-proxy image
# as a grc.io/kubebuilder registry image and pushes it (and all constituent images).
# This script cannot be inlined due to:
# https://github.com/GoogleCloudPlatform/cloud-build-local/issues/129

set -eu

SOURCE_IMAGE_TAG="quay.io/brancz/kube-rbac-proxy:${KUBE_RBAC_PROXY_VERSION}"
TARGET_IMAGE_TAG="gcr.io/kubebuilder/kube-rbac-proxy:${KUBE_RBAC_PROXY_VERSION}"

# Each arch to pull an image for.
declare ARCHES
ARCHES=( amd64 arm64 ppc64le s390x )

declare IMAGES
for a in ${ARCHES[@]}; do
  docker pull "${SOURCE_IMAGE_TAG}-$a"
  docker tag "${SOURCE_IMAGE_TAG}-$a" "${TARGET_IMAGE_TAG}-$a"
  # These images must exist remotely to build a manifest list.
  docker push "${TARGET_IMAGE_TAG}-$a"
  # weird syntax for bash<4.4
  IMAGES=( ${IMAGES[@]+"${IMAGES[@]}"} "${TARGET_IMAGE_TAG}-$a" )
done

# If $TARGET_IMAGE_TAG exists, `manifest create` will fail.
docker manifest rm "$TARGET_IMAGE_TAG" || true
docker manifest create "$TARGET_IMAGE_TAG" ${IMAGES[@]}
docker manifest push "$TARGET_IMAGE_TAG"
