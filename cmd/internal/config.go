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

package internal

import (
	"fmt"
	"log"
	"os"

	"sigs.k8s.io/kubebuilder/internal/config"
)

const (
	noticeColor = "\033[1;36m%s\033[0m"
)

// ConfiguredAndV1 returns true if the project is already configured and it is v1
func ConfiguredAndV1() bool {
	projectConfig, err := config.Read()

	if os.IsNotExist(err) {
		return false
	}

	if err != nil {
		log.Fatalf("failed to read the configuration file: %v", err)
	}

	return projectConfig.IsV1()
}

func PrintV1DeprecationWarning() {
	fmt.Printf(noticeColor, "[Deprecation Notice] The v1 projects are deprecated and will not be supported beyond "+
		"Feb 1, 2020.\nSee how to upgrade your project to v2: https://book.kubebuilder.io/migration/guide.html\n")
}
