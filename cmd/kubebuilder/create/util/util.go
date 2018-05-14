/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/markbates/inflect"
	"github.com/spf13/cobra"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
)

var (
	GroupName, KindName, VersionName, ResourceName, Copyright string
	AllowPluralKind                                           bool
)

func ValidateResourceFlags() {
	util.GetDomain()
	if len(GroupName) == 0 {
		log.Fatalf("Must specify --group")
	}
	if len(VersionName) == 0 {
		log.Fatalf("Must specify --version")
	}
	if len(KindName) == 0 {
		log.Fatal("Must specify --kind")
	}

	rs := inflect.NewDefaultRuleset()
	if len(ResourceName) == 0 {
		if !AllowPluralKind && rs.Pluralize(KindName) == KindName && rs.Singularize(KindName) != KindName {
			log.Fatalf("Client code generation expects singular --kind (e.g. %s)."+
				"Or to be run with --pural-kind=true.", rs.Singularize(KindName))
		}
		ResourceName = rs.Pluralize(strings.ToLower(KindName))
	}

	groupMatch := regexp.MustCompile("^[a-z]+$")
	if !groupMatch.MatchString(GroupName) {
		log.Fatalf("--group must match regex ^[a-z]+$ but was (%s)", GroupName)
	}
	versionMatch := regexp.MustCompile("^v\\d+(alpha\\d+|beta\\d+)*$")
	if !versionMatch.MatchString(VersionName) {
		log.Fatalf(
			"--version has bad format. must match ^v\\d+(alpha\\d+|beta\\d+)*$.  "+
				"e.g. v1alpha1,v1beta1,v1 but was (%s)", VersionName)
	}

	kindMatch := regexp.MustCompile("^[A-Z]+[A-Za-z0-9]*$")
	if !kindMatch.MatchString(KindName) {
		log.Fatalf("--kind must match regex ^[A-Z]+[A-Za-z0-9]*$ but was (%s)", KindName)
	}
}

func RegisterResourceFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&GroupName, "group", "", "name of the API group.  **Must be single lowercase word (match ^[a-z]+$)**.")
	cmd.Flags().StringVar(&VersionName, "version", "", "name of the API version.  **must match regex v\\d+(alpha\\d+|beta\\d+)** e.g. v1, v1beta1, v1alpha1")
	cmd.Flags().StringVar(&KindName, "kind", "", "name of the API kind.  **Must be CamelCased (match ^[A-Z]+[A-Za-z0-9]*$)**")
	cmd.Flags().StringVar(&ResourceName, "resource", "", "optional name of the API resource, defaults to the plural name of the lowercase kind")
}

func RegisterCopyrightFlag(cmd *cobra.Command) {
	cmd.Flags().StringVar(&Copyright, "copyright", filepath.Join("hack", "boilerplate.go.txt"), "Location of copyright boilerplate file.")
}
