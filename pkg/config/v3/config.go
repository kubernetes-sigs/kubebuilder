/*
Copyright 2021 The Kubernetes Authors.

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

package v3

import (
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

// Version is the config.Version for project configuration 3
var Version = config.Version{Number: 3}

// stringSlice is a []string but that can also be unmarshalled from a single string,
// which is introduced as the first and only element of the slice
// It is used to offer backwards compatibility as the field used to be a string.
type stringSlice []string

func (ss *stringSlice) UnmarshalJSON(b []byte) error {
	if b[0] == '[' {
		var sl []string
		if err := yaml.Unmarshal(b, &sl); err != nil {
			return err
		}
		*ss = sl
		return nil
	}

	var st string
	if err := yaml.Unmarshal(b, &st); err != nil {
		return err
	}
	*ss = stringSlice{st}
	return nil
}

type cfg struct {
	// Version
	Version config.Version `json:"version"`

	// String fields
	Domain      string      `json:"domain,omitempty"`
	Repository  string      `json:"repo,omitempty"`
	Name        string      `json:"projectName,omitempty"`
	PluginChain stringSlice `json:"layout,omitempty"`

	// Boolean fields
	MultiGroup      bool `json:"multigroup,omitempty"`
	ComponentConfig bool `json:"componentConfig,omitempty"`

	// Resources
	Resources []resource.Resource `json:"resources,omitempty"`

	// Plugins
	Plugins pluginConfigs `json:"plugins,omitempty"`
}

// pluginConfigs holds a set of arbitrary plugin configuration objects mapped by plugin key.
type pluginConfigs map[string]pluginConfig

// pluginConfig is an arbitrary plugin configuration object.
type pluginConfig interface{}

// New returns a new config.Config
func New() config.Config {
	return &cfg{Version: Version}
}

func init() {
	config.Register(Version, New)
}

// GetVersion implements config.Config
func (c cfg) GetVersion() config.Version {
	return c.Version
}

// GetDomain implements config.Config
func (c cfg) GetDomain() string {
	return c.Domain
}

// SetDomain implements config.Config
func (c *cfg) SetDomain(domain string) error {
	c.Domain = domain
	return nil
}

// GetRepository implements config.Config
func (c cfg) GetRepository() string {
	return c.Repository
}

// SetRepository implements config.Config
func (c *cfg) SetRepository(repository string) error {
	c.Repository = repository
	return nil
}

// GetProjectName implements config.Config
func (c cfg) GetProjectName() string {
	return c.Name
}

// SetProjectName implements config.Config
func (c *cfg) SetProjectName(name string) error {
	c.Name = name
	return nil
}

// GetLayout implements config.Config
func (c cfg) GetPluginChain() []string {
	return c.PluginChain
}

// SetLayout implements config.Config
func (c *cfg) SetPluginChain(pluginChain []string) error {
	c.PluginChain = pluginChain
	return nil
}

// IsMultiGroup implements config.Config
func (c cfg) IsMultiGroup() bool {
	return c.MultiGroup
}

// SetMultiGroup implements config.Config
func (c *cfg) SetMultiGroup() error {
	c.MultiGroup = true
	return nil
}

// ClearMultiGroup implements config.Config
func (c *cfg) ClearMultiGroup() error {
	c.MultiGroup = false
	return nil
}

// IsComponentConfig implements config.Config
func (c cfg) IsComponentConfig() bool {
	return c.ComponentConfig
}

// SetComponentConfig implements config.Config
func (c *cfg) SetComponentConfig() error {
	c.ComponentConfig = true
	return nil
}

// ClearComponentConfig implements config.Config
func (c *cfg) ClearComponentConfig() error {
	c.ComponentConfig = false
	return nil
}

// ResourcesLength implements config.Config
func (c cfg) ResourcesLength() int {
	return len(c.Resources)
}

// HasResource implements config.Config
func (c cfg) HasResource(gvk resource.GVK) bool {
	for _, res := range c.Resources {
		if gvk.IsEqualTo(res.GVK) {
			return true
		}
	}

	return false
}

// GetResource implements config.Config
func (c cfg) GetResource(gvk resource.GVK) (resource.Resource, error) {
	for _, res := range c.Resources {
		if gvk.IsEqualTo(res.GVK) {
			r := res.Copy()

			// Plural is only stored if irregular, so if it is empty recover the regular form
			if r.Plural == "" {
				r.Plural = resource.RegularPlural(r.Kind)
			}

			return r, nil
		}
	}

	return resource.Resource{}, config.ResourceNotFoundError{GVK: gvk}
}

// GetResources implements config.Config
func (c cfg) GetResources() ([]resource.Resource, error) {
	resources := make([]resource.Resource, 0, len(c.Resources))
	for _, res := range c.Resources {
		r := res.Copy()

		// Plural is only stored if irregular, so if it is empty recover the regular form
		if r.Plural == "" {
			r.Plural = resource.RegularPlural(r.Kind)
		}

		resources = append(resources, r)
	}

	return resources, nil
}

// AddResource implements config.Config
func (c *cfg) AddResource(res resource.Resource) error {
	// As res is passed by value it is already a shallow copy, but we need to make a deep copy
	res = res.Copy()

	// Plural is only stored if irregular
	if res.Plural == resource.RegularPlural(res.Kind) {
		res.Plural = ""
	}

	if !c.HasResource(res.GVK) {
		c.Resources = append(c.Resources, res)
	}
	return nil
}

// UpdateResource implements config.Config
func (c *cfg) UpdateResource(res resource.Resource) error {
	// As res is passed by value it is already a shallow copy, but we need to make a deep copy
	res = res.Copy()

	// Plural is only stored if irregular
	if res.Plural == resource.RegularPlural(res.Kind) {
		res.Plural = ""
	}

	for i, r := range c.Resources {
		if res.GVK.IsEqualTo(r.GVK) {
			return c.Resources[i].Update(res)
		}
	}

	c.Resources = append(c.Resources, res)
	return nil
}

// HasGroup implements config.Config
func (c cfg) HasGroup(group string) bool {
	// Return true if the target group is found in the tracked resources
	for _, r := range c.Resources {
		if strings.EqualFold(group, r.Group) {
			return true
		}
	}

	// Return false otherwise
	return false
}

// ListCRDVersions implements config.Config
func (c cfg) ListCRDVersions() []string {
	// Make a map to remove duplicates
	versionSet := make(map[string]struct{})
	for _, r := range c.Resources {
		if r.API != nil && r.API.CRDVersion != "" {
			versionSet[r.API.CRDVersion] = struct{}{}
		}
	}

	// Convert the map into a slice
	versions := make([]string, 0, len(versionSet))
	for version := range versionSet {
		versions = append(versions, version)
	}
	return versions
}

// ListWebhookVersions implements config.Config
func (c cfg) ListWebhookVersions() []string {
	// Make a map to remove duplicates
	versionSet := make(map[string]struct{})
	for _, r := range c.Resources {
		if r.Webhooks != nil && r.Webhooks.WebhookVersion != "" {
			versionSet[r.Webhooks.WebhookVersion] = struct{}{}
		}
	}

	// Convert the map into a slice
	versions := make([]string, 0, len(versionSet))
	for version := range versionSet {
		versions = append(versions, version)
	}
	return versions
}

// DecodePluginConfig implements config.Config
func (c cfg) DecodePluginConfig(key string, configObj interface{}) error {
	if len(c.Plugins) == 0 {
		return config.PluginKeyNotFoundError{Key: key}
	}

	// Get the object blob by key and unmarshal into the object.
	if pluginConfig, hasKey := c.Plugins[key]; hasKey {
		b, err := yaml.Marshal(pluginConfig)
		if err != nil {
			return fmt.Errorf("failed to convert extra fields object to bytes: %w", err)
		}
		if err := yaml.Unmarshal(b, configObj); err != nil {
			return fmt.Errorf("failed to unmarshal extra fields object: %w", err)
		}
		return nil
	}

	return config.PluginKeyNotFoundError{Key: key}
}

// EncodePluginConfig will return an error if used on any project version < v3.
func (c *cfg) EncodePluginConfig(key string, configObj interface{}) error {
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

// Marshal implements config.Config
func (c cfg) MarshalYAML() ([]byte, error) {
	for i, r := range c.Resources {
		// If API is empty, omit it (prevents `api: {}`).
		if r.API != nil && r.API.IsEmpty() {
			c.Resources[i].API = nil
		}
		// If Webhooks is empty, omit it (prevents `webhooks: {}`).
		if r.Webhooks != nil && r.Webhooks.IsEmpty() {
			c.Resources[i].Webhooks = nil
		}
	}

	content, err := yaml.Marshal(c)
	if err != nil {
		return nil, config.MarshalError{Err: err}
	}

	return content, nil
}

// Unmarshal implements config.Config
func (c *cfg) UnmarshalYAML(b []byte) error {
	if err := yaml.UnmarshalStrict(b, c); err != nil {
		return config.UnmarshalError{Err: err}
	}

	return nil
}
