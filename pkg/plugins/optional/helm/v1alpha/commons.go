/*
Copyright 2024 The Kubernetes Authors.

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

// Package v1alpha implements the Helm chart scaffolding plugin for Kubebuilder.
//
// Deprecated: The helm/v1alpha package is deprecated. Use [helm/v2alpha] instead.
// The new helm/v2-alpha plugin replaces the deprecated helm/v1-alpha and brings major improvements in
// flexibility and maintainability, championing changes driven by community feedback. Chart values are
// now better exposed, enabling easier customization and addressing long-standing issues.
// See [Helm Plugin (helm/v2-alpha)].
//
// This package is kept for compatibility and will eventually be removed.
//
// [helm/v2alpha]: https://pkg.go.dev/sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha
// [Helm Plugin (helm/v2-alpha)]: https://book.kubebuilder.io/plugins/available/helm-v2-alpha
package v1alpha

import (
	"errors"
	"fmt"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

func insertPluginMetaToConfig(target config.Config, cfg pluginConfig) error {
	key := plugin.GetPluginKeyForConfig(target.GetPluginChain(), Plugin{})
	canonicalKey := plugin.KeyFor(Plugin{})

	if err := target.DecodePluginConfig(key, &cfg); err != nil {
		switch {
		case errors.As(err, &config.UnsupportedFieldError{}):
			return nil
		case errors.As(err, &config.PluginKeyNotFoundError{}):
			if key != canonicalKey {
				if err2 := target.DecodePluginConfig(canonicalKey, &cfg); err2 != nil {
					if errors.As(err2, &config.UnsupportedFieldError{}) {
						return nil
					}
					if !errors.As(err2, &config.PluginKeyNotFoundError{}) {
						return fmt.Errorf("error decoding plugin configuration: %w", err2)
					}
				}
			}
		default:
			return fmt.Errorf("error decoding plugin configuration: %w", err)
		}
	}

	if err := target.EncodePluginConfig(key, cfg); err != nil {
		return fmt.Errorf("error encoding plugin config: %w", err)
	}

	return nil
}
