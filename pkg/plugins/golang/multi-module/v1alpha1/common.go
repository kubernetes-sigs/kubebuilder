/*
Copyright 2022 The Kubernetes Authors.

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

package v1alpha1

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
)

func tidyGoModForAPI(apiPath string) error {
	return util.RunInDir(apiPath, func() error {
		if err := util.RunCmd(
			"update dependencies in "+apiPath, "go", "mod", "tidy"); err != nil {
			return err
		}
		return nil
	})
}

func getAPIPath(isMultiGroup bool) string {
	path := ""
	if isMultiGroup {
		path = filepath.Join("apis")
	} else {
		path = filepath.Join("api")
	}
	return path
}
