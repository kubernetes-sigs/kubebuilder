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

package v1alpha

import (
	"errors"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
)

func InsertPluginMetaToConfig(target config.Config, cfg pluginConfig) error {
	err := target.DecodePluginConfig(pluginKey, cfg)
	if !errors.As(err, &config.UnsupportedFieldError{}) {

		if err != nil && !errors.As(err, &config.PluginKeyNotFoundError{}) {
			return err
		}

		if err = target.EncodePluginConfig(pluginKey, cfg); err != nil {
			return err
		}

	}

	return nil
}
