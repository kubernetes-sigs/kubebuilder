#!/bin/bash

# Copyright 2021 The Kubernetes Authors.
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

PLUGIN_NAME="sampleexternalplugin"
PLUGIN_VERSION="v1"
PLUGIN_BINARY="./bin/${PLUGIN_NAME}"

if [[ ! -f "${PLUGIN_BINARY}" ]]; then
  echo "Plugin binary not found at ${PLUGIN_BINARY}"
  echo "Make sure you run: make build"
  exit 1
fi

# Detect OS and set plugin destination path
if [[ "$OSTYPE" == "darwin"* ]]; then
  PLUGIN_DEST="$HOME/Library/Application Support/kubebuilder/plugins/${PLUGIN_NAME}/${PLUGIN_VERSION}"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
  PLUGIN_DEST="$HOME/.config/kubebuilder/plugins/${PLUGIN_NAME}/${PLUGIN_VERSION}"
else
  echo "Unsupported OS: $OSTYPE"
  exit 1
fi

mkdir -p "${PLUGIN_DEST}"

cp "${PLUGIN_BINARY}" "${PLUGIN_DEST}/${PLUGIN_NAME}"
chmod +x "${PLUGIN_DEST}/${PLUGIN_NAME}"

echo "Plugin installed at:"
echo "${PLUGIN_DEST}/${PLUGIN_NAME}"
