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

	// ProjectName is the name of this controller project set on initialization.
	ProjectName string `json:"projectName,omitempty"`

	// Resources tracks scaffolded resources in the project
	// This info is tracked only in project with version 2
	Resources []ResourceData `json:"resources,omitempty"`

	// Multigroup tracks if the project has more than one group
	MultiGroup bool `json:"multigroup,omitempty"`

	// ComponentConfig tracks if the project uses a config file for configuring
	// the ctrl.Manager
	ComponentConfig bool `json:"componentConfig,omitempty"`

	// Layout contains a key specifying which plugin created a project.
	Layout string `json:"layout,omitempty"`

	// Plugins holds plugin-specific configs mapped by plugin key. These configs should be
	// encoded/decoded using EncodePluginConfig/DecodePluginConfig, respectively.
	Plugins PluginConfigs `json:"plugins,omitempty"`
}

// PluginConfigs holds a set of arbitrary plugin configuration objects mapped by plugin key.
type PluginConfigs map[string]pluginConfig

// pluginConfig is an arbitrary plugin configuration object.
type pluginConfig interface{}

// IsV2 returns true if it is a v2 project
func (c Config) IsV2() bool {
	return c.Version == Version2
}

// IsV3 returns true if it is a v3 project
func (c Config) IsV3() bool {
	return c.Version == Version3Alpha
}

// GetResource returns the GKV if the resource is found
func (c Config) GetResource(target ResourceData) *ResourceData {
	// Return true if the target resource is found in the tracked resources
	for _, r := range c.Resources {
		if r.isGVKEqualTo(target) {
			return &r
		}
	}
	return nil
}

// UpdateResources either adds gvk to the tracked set or, if the resource already exists,
// updates the the equivalent resource in the set.
func (c *Config) UpdateResources(resource ResourceData) {
	// If the resource already exists, update it.
	for i, r := range c.Resources {
		if r.isGVKEqualTo(resource) {
			c.Resources[i].merge(resource)
			return
		}
	}

	// The resource does not exist, append the resource to the tracked ones.
	c.Resources = append(c.Resources, resource)
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

// IsCRDVersionCompatible returns true if crdVersion can be added to the existing set of CRD versions.
func (c Config) IsCRDVersionCompatible(crdVersion string) bool {
	return c.resourceAPIVersionCompatible("crd", crdVersion)
}

// IsWebhookVersionCompatible returns true if webhookVersion can be added to the existing set of Webhook versions.
func (c Config) IsWebhookVersionCompatible(webhookVersion string) bool {
	return c.resourceAPIVersionCompatible("webhook", webhookVersion)
}

// resourceAPIVersionCompatible returns true if version can be added to the existing set of versions
// for a given verType.
func (c Config) resourceAPIVersionCompatible(verType, version string) bool {
	for _, res := range c.Resources {
		var currVersion string
		switch verType {
		case "crd":
			if res.API != nil {
				currVersion = res.API.CRDVersion
			}
		case "webhook":
			if res.Webhooks != nil {
				currVersion = res.Webhooks.WebhookVersion
			}
		}
		if currVersion != "" && version != currVersion {
			return false
		}
	}
	return true
}

// ResourceData contains information about scaffolded resources
type ResourceData struct {
	Group   string `json:"group,omitempty"`
	Version string `json:"version,omitempty"`
	Kind    string `json:"kind,omitempty"`

	// API holds the API data
	API *API `json:"api,omitempty"`

	// Webhooks holds the Webhooks data
	Webhooks *Webhooks `json:"webhooks,omitempty"`
}

// API contains information about scaffolded APIs
type API struct {
	// CRDVersion holds the CustomResourceDefinition API version used for the ResourceData.
	CRDVersion string `json:"crdVersion,omitempty"`
}

// Webhooks contains information about scaffolded webhooks
type Webhooks struct {
	// WebhookVersion holds the {Validating,Mutating}WebhookConfiguration API version used for the Options.
	WebhookVersion string `json:"webhookVersion,omitempty"`
}

// isGVKEqualTo compares it with another resource
func (r ResourceData) isGVKEqualTo(other ResourceData) bool {
	return r.Group == other.Group &&
		r.Version == other.Version &&
		r.Kind == other.Kind
}

// merge combines fields of two GVKs that have matching group, version, and kind,
// favoring the receiver's values.
func (r *ResourceData) merge(other ResourceData) {
	if other.Webhooks != nil {
		if r.Webhooks == nil {
			r.Webhooks = other.Webhooks
		} else {
			r.Webhooks.merge(other.Webhooks)
		}
	}

	if other.API != nil {
		if r.API == nil {
			r.API = other.API
		} else {
			r.API.merge(other.API)
		}
	}
}

// merge compares it with another webhook by setting each webhook type individually so existing values are
// not overwritten.
func (w *Webhooks) merge(other *Webhooks) {
	if w.WebhookVersion == "" && other.WebhookVersion != "" {
		w.WebhookVersion = other.WebhookVersion
	}
}

// merge compares it with another api by setting each api type individually so existing values are
// not overwritten.
func (a *API) merge(other *API) {
	if a.CRDVersion == "" && other.CRDVersion != "" {
		a.CRDVersion = other.CRDVersion
	}
}

// Marshal returns the bytes of c.
func (c Config) Marshal() ([]byte, error) {
	// Ignore extra fields at first.
	cfg := c
	cfg.Plugins = nil

	// Ignore some fields if v2.
	if cfg.IsV2() {
		for i := range cfg.Resources {
			cfg.Resources[i].API = nil
			cfg.Resources[i].Webhooks = nil
		}
	}

	for i, r := range cfg.Resources {
		// If API is empty, omit it (prevents `api: {}`).
		if r.API != nil && r.API.CRDVersion == "" {
			cfg.Resources[i].API = nil
		}
		// If Webhooks is empty, omit it (prevents `webhooks: {}`).
		if r.Webhooks != nil && r.Webhooks.WebhookVersion == "" {
			cfg.Resources[i].Webhooks = nil
		}
	}

	content, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("error marshalling project configuration: %v", err)
	}

	// Empty config strings are "{}" due to the map field.
	if strings.TrimSpace(string(content)) == "{}" {
		content = []byte{}
	}

	// Append extra fields to put them at the config's bottom.
	if len(c.Plugins) != 0 {
		// Unless the project version is v2 which does not support a plugins field.
		if cfg.IsV2() {
			return nil, fmt.Errorf("error marshalling project configuration: plugin field found for v2")
		}

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

	// Project versions < v3 do not support a plugins field.
	if !c.IsV3() {
		c.Plugins = nil
	}
	return nil
}

// EncodePluginConfig encodes a config object into c by overwriting the existing
// object stored under key. This method is intended to be used for custom
// configuration objects, which were introduced in project version 3-alpha.
// EncodePluginConfig will return an error if used on any project version < v3.
func (c *Config) EncodePluginConfig(key string, configObj interface{}) error {
	// Short-circuit project versions < v3.
	if !c.IsV3() {
		return fmt.Errorf("project versions < v3 do not support extra fields")
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
		c.Plugins = make(map[string]pluginConfig)
	}
	c.Plugins[key] = fields
	return nil
}

// DecodePluginConfig decodes a plugin config stored in c into configObj, which must be a pointer
// This method is intended to be used for custom configuration objects, which were introduced
// in project version 3-alpha. EncodePluginConfig will return an error if used on any project version < v3.
func (c Config) DecodePluginConfig(key string, configObj interface{}) error {
	// Short-circuit project versions < v3.
	if !c.IsV3() {
		return fmt.Errorf("project versions < v3 do not support extra fields")
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
