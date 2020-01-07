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

#
# Script to generate Gopkg.toml file with override stanzas 
# Script assumes you have run 'dep ensure' on the project and it is
# in compilable state.
#
# Notes about generating override stanza:
# Create a project using Kubebuilder
# Remove all the override stanzas except the k8s one.
# Run 'dep ensure' to update the dependencies and then
# run this script.
# Save the output of the script and copy its content to replace the value
# depManifestOverride variable in cmd/kubebuilder/initproject/dep_manifest.go
#

########################################################################################################
# NOTE: It is deprecated since is valid just for version 1                                             #
########################################################################################################

tmp_file="/tmp/dep.txt"
dep status -json|jq '.[]|.ProjectRoot,.Revision, .Version' > $tmp_file
while read name ; do 
	read revision; read version; 
	# strip quotes
	myver=$(echo "$version" | tr -d '"')
	myname=$(echo "$name"|tr -d '"')
	if [ "$myname" = "sigs.k8s.io/kubebuilder" ]; then
		continue
	fi
	if [ "$myver" = "branch master" ] || [ -z "$myver" ]; then
		printf "\n[[override]]\nname=$name\nrevision=$revision\n" ; 
	else
		printf "\n[[override]]\nname=$name\nversion=$version\n" ; 
	fi
done < $tmp_file

cat << EOF

[[override]]
name = "sigs.k8s.io/kubebuilder"
{{ if eq .Version "unknown" -}}
branch = "master"
{{ else -}}
version = "{{.Version}}"
{{ end }}
EOF
