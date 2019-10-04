#!/usr/bin/env bash

#  Copyright 2018 The Kubernetes Authors.
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

source $(dirname ${BASH_SOURCE})/common.sh

setup_envs

header_text "running go test"

go test ./pkg/... -parallel 4

header_text "running coverage"

# Verify no coverage regressions have been introduced.  Remove the exception list from here
# once the coverage has been brought back up
if [[ ! $(go test ./pkg/...  -coverprofile cover.out -parallel 4 | grep -v "coverage: 100.0% of statements" | grep "controller-runtime/pkg " | grep -v "controller-runtime/pkg \|controller-runtime/pkg/recorder \|pkg/admission/certprovisioner \|pkg/internal/admission \|pkg/cache\|pkg/client \|pkg/event \|pkg/client/config \|pkg/controller/controllertest \|pkg/reconcile/reconciletest \|pkg/test ") ]]; then
echo "ok"
else
go test ./pkg/...  -coverprofile cover.out -parallel 4 | grep -v "coverage: 100.0% of statements" | grep "controller-runtime/pkg " | grep -v "controller-runtime/pkg \|controller-runtime/pkg/recorder \|pkg/admission/certprovisioner \|pkg/internal/admission \|pkg/cache\|pkg/client \|pkg/event \|pkg/client/config \|pkg/controller/controllertest \|pkg/reconcile/reconciletest \|pkg/test "
echo "missing test coverage"
exit 1
fi
