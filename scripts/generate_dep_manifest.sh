#!/bin/bash

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

tmp_file="/tmp/dep.txt"
dep status -json|jq '.[]|.ProjectRoot,.Revision, .Version' > $tmp_file
while read name ; do 
	read revision; read version; 
	# strip quotes
	myver=$(echo "$version" | tr -d '"')
	myname=$(echo "$name"|tr -d '"')
	if [ "$myname" = "github.com/kubernetes-sigs/kubebuilder" ]; then
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
name = "github.com/kubernetes-sigs/kubebuilder"
{{ if eq .Version "unknown" -}}
branch = "master"
{{ else -}}
version = "{{.Version}}"
{{ end }}
EOF
