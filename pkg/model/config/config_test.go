/*
Copyright 2020 The Kubernetes Authors.

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

package config

import (
	"reflect"
	"testing"
)

func TestEncodeDecodePluginConfig(t *testing.T) {
	// Test plugin config. Don't want to export this config, but need it to
	// be accessible by unmarshallers.
	type PluginConfig struct {
		Data1 string `json:"data-1"`
		Data2 string `json:"data-2"`
	}

	cases := []struct {
		description     string
		config          Config
		key             string
		pluginConfigObj interface{}
		expConfig       Config
		wantErr         bool
	}{
		{
			description: "config version 1",
			config:      Config{Version: Version1},
			wantErr:     true,
		},
		{
			description: "config version 1 with extra fields",
			config:      Config{Version: Version1},
			key:         "plugin-x",
			pluginConfigObj: map[string]interface{}{
				"plugin-x": "single plugin datum",
			},
			wantErr: true,
		},
		{
			description:     "config version 2",
			key:             "plugin-x",
			pluginConfigObj: PluginConfig{},
			config:          Config{Version: Version2},
			expConfig: Config{
				Version: Version2,
				Plugins: map[string]interface{}{
					"plugin-x": map[string]interface{}{
						"data-1": "",
						"data-2": "",
					},
				},
			},
		},
		{
			description: "config version 2 with extra fields as struct",
			config:      Config{Version: Version2},
			key:         "plugin-x",
			pluginConfigObj: PluginConfig{
				"plugin value 1",
				"plugin value 2",
			},
			expConfig: Config{
				Version: Version2,
				Plugins: map[string]interface{}{
					"plugin-x": map[string]interface{}{
						"data-1": "plugin value 1",
						"data-2": "plugin value 2",
					},
				},
			},
		},
	}

	for _, c := range cases {
		// Test EncodePluginConfig
		err := c.config.EncodePluginConfig(c.key, c.pluginConfigObj)
		if err != nil {
			if !c.wantErr {
				t.Errorf("%s: expected EncodePluginConfig to succeed, got error: %s", c.description, err)
			}
		} else if c.wantErr {
			t.Errorf("%s: expected EncodePluginConfig to fail, got no error", c.description)
		} else if !reflect.DeepEqual(c.expConfig, c.config) {
			t.Errorf("%s: compare encoded configs\nexpected:\n%#v\n\nreturned:\n%#v",
				c.description, c.expConfig, c.config)
		}

		// Test DecodePluginConfig
		obj := PluginConfig{}
		err = c.config.DecodePluginConfig(c.key, &obj)
		if err != nil {
			if !c.wantErr {
				t.Errorf("%s: expected DecodePluginConfig to succeed, got error: %s", c.description, err)
			}
		} else if c.wantErr {
			t.Errorf("%s: expected DecodePluginConfig to fail, got no error", c.description)
		} else if !reflect.DeepEqual(c.pluginConfigObj, obj) {
			t.Errorf("%s: compare decoded extra fields objs\nexpected:\n%#v\n\nreturned:\n%#v",
				c.description, c.pluginConfigObj, obj)
		}
	}
}
