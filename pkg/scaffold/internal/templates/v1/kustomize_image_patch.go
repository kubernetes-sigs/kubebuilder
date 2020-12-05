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

package v1

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v2/pkg/model/file"
)

var _ file.Template = &KustomizeImagePatch{}

// KustomizeImagePatch scaffolds the patch file for customizing image URL
// manifest file for manager resource.
type KustomizeImagePatch struct {
	file.TemplateMixin

	// ImageURL to use for controller image in manager's manifest.
	ImageURL string
}

// SetTemplateDefaults implements input.Template
func (f *KustomizeImagePatch) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "default", "manager_image_patch.yaml")
	}

	f.TemplateBody = kustomizeImagePatchTemplate

	f.IfExistsAction = file.Error

	if f.ImageURL == "" {
		f.ImageURL = "IMAGE_URL"
	}

	return nil
}

const kustomizeImagePatchTemplate = `apiVersion: apps/v1
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
