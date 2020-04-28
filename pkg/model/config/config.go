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
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"
)

// Scaffolding versions
const (
	Version1      = "1"
	Version2      = "2"
	Version3Alpha = "3-alpha"
)

// Config is the unmarshalled representation of the configuration file
type Config struct {
	// Version is the project version, defaults to "1" (backwards compatibility)
	Version string `json:"version,omitempty"`

	// Domain is the domain associated with the project and used for API groups
	Domain string `json:"domain,omitempty"`

	// Repo is the go package name of the project root
	Repo string `json:"repo,omitempty"`

	// Resources tracks scaffolded resources in the project
	// This info is tracked only in project with version 2
	Resources []GVK `json:"resources,omitempty"`

	// Multigroup tracks if the project has more than one group
	MultiGroup bool `json:"multigroup,omitempty"`

	// Layout contains a key specifying which plugin created a project.
	Layout string `json:"layout,omitempty"`

	// Plugins is an arbitrary YAML blob that can be used by external
	// plugins for plugin-specific configuration.
	Plugins map[string]interface{} `json:"plugins,omitempty"`
}

// IsV1 returns true if it is a v1 project
func (c Config) IsV1() bool {
	return c.Version == Version1
}

// IsV2 returns true if it is a v2 project
func (c Config) IsV2() bool {
	return c.Version == Version2
}

// IsV3 returns true if it is a v3 project
func (c Config) IsV3() bool {
	return c.Version == Version3Alpha
}

// HasResource returns true if API resource is already tracked
func (c Config) HasResource(target GVK) bool {
	// Return true if the target resource is found in the tracked resources
	for _, r := range c.Resources {
		if r.isEqualTo(target) {
			return true
		}
	}

	// Return false otherwise
	return false
}

// AddResource appends the provided resource to the tracked ones
// It returns if the configuration was modified
// NOTE: in v1 resources are not tracked, so we return false
func (c *Config) AddResource(gvk GVK) bool {
	// Short-circuit v1
	if c.IsV1() {
		return false
	}

	// No-op if the resource was already tracked, return false
	if c.HasResource(gvk) {
		return false
	}

	// Append the resource to the tracked ones, return true
	c.Resources = append(c.Resources, gvk)
	return true
}

// HasGroup returns true if group is already tracked
func (c Config) HasGroup(group string) bool {
	// Return true if the target group is found in the tracked resources
	for _, r := range c.Resources {
		if strings.EqualFold(group, r.Group) {
			return true
		}
	}

	// Return false otherwise
	return false
}

// GVK contains information about scaffolded resources
type GVK struct {
	Group   string `json:"group,omitempty"`
	Version string `json:"version,omitempty"`
	Kind    string `json:"kind,omitempty"`
}

// isEqualTo compares it with another resource
func (r GVK) isEqualTo(other GVK) bool {
	return r.Group == other.Group &&
		r.Version == other.Version &&
		r.Kind == other.Kind
}

// Marshal returns the bytes of c.
func (c Config) Marshal() ([]byte, error) {
	// Ignore extra fields at first.
	cfg := c
	cfg.Plugins = nil
	content, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("error marshalling project configuration: %v", err)
	}
	// Empty config strings are "{}" due to the map field.
	if strings.TrimSpace(string(content)) == "{}" {
		content = []byte{}
	}
	// Append extra fields to put them at the config's bottom, unless the
	// project is v1 which does not support extra fields.
	if !cfg.IsV1() && len(c.Plugins) != 0 {
		pluginConfigBytes, err := yaml.Marshal(Config{Plugins: c.Plugins})
		if err != nil {
			return nil, fmt.Errorf("error marshalling project configuration extra fields: %v", err)
		}
		content = append(content, pluginConfigBytes...)
	}
	return content, nil
}

// Unmarshal unmarshals the bytes of a Config into c.
func (c *Config) Unmarshal(b []byte) error {
	if err := yaml.UnmarshalStrict(b, c); err != nil {
		return fmt.Errorf("error unmarshalling project configuration: %v", err)
	}
	// v1 projects do not support extra fields.
	if c.IsV1() {
		c.Plugins = nil
	}
	return nil
}

// EncodePluginConfig encodes a config object into c by overwriting the existing
// object stored under key. This method is intended to be used for custom
// configuration objects.
func (c *Config) EncodePluginConfig(key string, configObj interface{}) error {
	// Short-circuit v1
	if c.IsV1() {
		return fmt.Errorf("v1 project configs do not have extra fields")
	}

	// Get object's bytes and set them under key in extra fields.
	b, err := yaml.Marshal(configObj)
	if err != nil {
		return fmt.Errorf("failed to convert %T object to bytes: %s", configObj, err)
	}
	var fields map[string]interface{}
	if err := yaml.Unmarshal(b, &fields); err != nil {
		return fmt.Errorf("failed to unmarshal %T object bytes: %s", configObj, err)
	}
	if c.Plugins == nil {
		c.Plugins = make(map[string]interface{})
	}
	c.Plugins[key] = fields
	return nil
}

// DecodePluginConfig decodes a plugin config stored in c into configObj. This
// method is intended to be used for custom configuration objects.
// configObj must be a pointer.
func (c Config) DecodePluginConfig(key string, configObj interface{}) error {
	// Short-circuit v1
	if c.IsV1() {
		return fmt.Errorf("v1 project configs do not have extra fields")
	}
	if len(c.Plugins) == 0 {
		return nil
	}

	// Get the object blob by key and unmarshal into the object.
	if pluginConfig, hasKey := c.Plugins[key]; hasKey {
		b, err := yaml.Marshal(pluginConfig)
		if err != nil {
			return fmt.Errorf("failed to convert extra fields object to bytes: %s", err)
		}
		if err := yaml.Unmarshal(b, configObj); err != nil {
			return fmt.Errorf("failed to unmarshal extra fields object: %s", err)
		}
	}
	return nil
}
