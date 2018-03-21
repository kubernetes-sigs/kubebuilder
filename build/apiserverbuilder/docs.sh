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
set -x

# Copy over the files from the host repo
export D=$DIR/$OUTPUT
mkdir -p $D

# Copy the api definitions
cp -r /host/repo/pkg $DIR/pkg

# Copy the docs
if [ -d "/host/repo/$OUTPUT" ]; then
    cp -r /host/repo/$OUTPUT/* $D
fi
if [ ! -d "$DIR/boilerplate.go.txt" ]; then
    touch $DIR/boilerplate.go.txt
else
    cp /host/repo/boilerplate.go.txt $DIR/boilerplate.go.txt
fi

cd $DIR

# Generate the artifacts
apiserver-boot init repo --domain $DOMAIN
apiserver-boot build generated clean
apiserver-boot build generated

# Generate the input .md files for the docs
go build -o bin/apiserver cmd/apiserver/main.go
bin/apiserver --etcd-servers=http://localhost:2379 --secure-port=9443 --print-openapi --delegated-auth=false > $OUTPUT/openapi-spec/swagger.json
gen-apidocs --build-operations=false --use-tags=false --allow-errors --config-dir=$OUTPUT

# Copy the input files to the host
if [ ! -d "/host/repo/$OUTPUT" ]; then
    mkdir -p /host/repo/$OUTPUT
fi
cp -r $OUTPUT/* /host/repo/$OUTPUT
