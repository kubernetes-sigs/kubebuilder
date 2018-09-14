/*
Copyright 2018 The Kubernetes Authors.

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

package project

import (
	"path/filepath"

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
)

var _ input.File = &KustomizeImagePatch{}

// KustomizeImagePatch scaffolds the patch file for customizing image URL
// manifest file for manager resource.
type KustomizeImagePatch struct {
	input.Input

	// ImageURL to use for controller image in manager's manifest.
	ImageURL string
}

// GetInput implements input.File
func (c *KustomizeImagePatch) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = filepath.Join("config", "default", "manager_image_patch.yaml")
	}
	if c.ImageURL == "" {
		c.ImageURL = "IMAGE_URL"
	}
	c.TemplateBody = kustomizeImagePatchTemplate
	c.Input.IfExistsAction = input.Error
	return c.Input, nil
}

var kustomizeImagePatchTemplate = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: controller-manager
  namespace: system
spec:
  template:
    spec:
      containers:
      # Change the value of image field below to your controller image URL
      - image: {{ .ImageURL }}
        name: manager
`
