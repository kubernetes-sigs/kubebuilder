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

package v3

import (
	"fmt"
	"reflect"
	"strings"

	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
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
			return fmt.Errorf("error unmarshalling string slice %q: %w", sl, err)
		}
		*ss = sl
		return nil
	}

	var st string
	if err := yaml.Unmarshal(b, &st); err != nil {
		return fmt.Errorf("error unmarshalling string %q: %w", st, err)
	}
	*ss = stringSlice{st}
	return nil
}

// Cfg defines the Project Config (PROJECT file)
type Cfg struct {
	// Version
	Version config.Version `json:"version"`

	// String fields
	Domain      string      `json:"domain,omitempty"`
	Repository  string      `json:"repo,omitempty"`
	Name        string      `json:"projectName,omitempty"`
	CliVersion  string      `json:"cliVersion,omitempty"`
	PluginChain stringSlice `json:"layout,omitempty"`

	// Boolean fields
	MultiGroup bool `json:"multigroup,omitempty"`
	Namespaced bool `json:"namespaced,omitempty"`

	// Resources
	Resources []resource.Resource `json:"resources,omitempty"`

	// Plugins
	Plugins pluginConfigs `json:"plugins,omitempty"`
}

// pluginConfigs holds a set of arbitrary plugin configuration objects mapped by plugin key.
type pluginConfigs map[string]pluginConfig

// pluginConfig is an arbitrary plugin configuration object.
type pluginConfig any

// New returns a new config.Config
func New() config.Config {
	return &Cfg{Version: Version}
}

func init() {
	config.Register(Version, New)
}

// GetVersion implements config.Config
func (c Cfg) GetVersion() config.Version {
	return c.Version
}

// GetCliVersion implements config.Config
func (c Cfg) GetCliVersion() string {
	return c.CliVersion
}

// SetCliVersion implements config.Config
func (c *Cfg) SetCliVersion(version string) error {
	c.CliVersion = version
	return nil
}

// GetDomain implements config.Config
func (c Cfg) GetDomain() string {
	return c.Domain
}

// SetDomain implements config.Config
func (c *Cfg) SetDomain(domain string) error {
	c.Domain = domain
	return nil
}

// GetRepository implements config.Config
func (c Cfg) GetRepository() string {
	return c.Repository
}

// SetRepository implements config.Config
func (c *Cfg) SetRepository(repository string) error {
	c.Repository = repository
	return nil
}

// GetProjectName implements config.Config
func (c Cfg) GetProjectName() string {
	return c.Name
}

// SetProjectName implements config.Config
func (c *Cfg) SetProjectName(name string) error {
	c.Name = name
	return nil
}

// GetPluginChain implements config.Config
func (c Cfg) GetPluginChain() []string {
	return c.PluginChain
}

// SetPluginChain implements config.Config
func (c *Cfg) SetPluginChain(pluginChain []string) error {
	c.PluginChain = pluginChain
	return nil
}

// IsMultiGroup implements config.Config
func (c Cfg) IsMultiGroup() bool {
	return c.MultiGroup
}

// SetMultiGroup implements config.Config
func (c *Cfg) SetMultiGroup() error {
	c.MultiGroup = true
	return nil
}

// ClearMultiGroup implements config.Config
func (c *Cfg) ClearMultiGroup() error {
	c.MultiGroup = false
	return nil
}

// IsNamespaced implements config.Config
func (c Cfg) IsNamespaced() bool {
	return c.Namespaced
}

// SetNamespaced implements config.Config
func (c *Cfg) SetNamespaced() error {
	c.Namespaced = true
	return nil
}

// ClearNamespaced implements config.Config
func (c *Cfg) ClearNamespaced() error {
	c.Namespaced = false
	return nil
}

// ResourcesLength implements config.Config
func (c Cfg) ResourcesLength() int {
	return len(c.Resources)
}

// HasResource implements config.Config
func (c Cfg) HasResource(gvk resource.GVK) bool {
	found := false
	for _, res := range c.Resources {
		if gvk.IsEqualTo(res.GVK) {
			found = true
			break
		}
	}

	return found
}

// GetResource implements config.Config
func (c Cfg) GetResource(gvk resource.GVK) (resource.Resource, error) {
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
func (c Cfg) GetResources() ([]resource.Resource, error) {
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
func (c *Cfg) AddResource(res resource.Resource) error {
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
func (c *Cfg) UpdateResource(res resource.Resource) error {
	// As res is passed by value it is already a shallow copy, but we need to make a deep copy
	res = res.Copy()

	// Plural is only stored if irregular
	if res.Plural == resource.RegularPlural(res.Kind) {
		res.Plural = ""
	}

	for i, r := range c.Resources {
		if res.IsEqualTo(r.GVK) {
			if err := c.Resources[i].Update(res); err != nil {
				return fmt.Errorf("failed to update resource %q: %w", res.GVK, err)
			}

			return nil
		}
	}

	c.Resources = append(c.Resources, res)
	return nil
}

// RemoveResource implements config.Config
func (c *Cfg) RemoveResource(gvk resource.GVK) error {
	indexToRemove := -1
	for i, r := range c.Resources {
		// Match by Group, Version, Kind (not domain, as core types may not have domain set)
		if r.Group == gvk.Group && r.Version == gvk.Version && r.Kind == gvk.Kind {
			indexToRemove = i
			break
		}
	}

	if indexToRemove == -1 {
		return fmt.Errorf("failed to remove resource: resource with GVK {%q %q %q} not found",
			gvk.Group, gvk.Version, gvk.Kind)
	}

	// Remove the resource by slicing around it
	c.Resources = append(c.Resources[:indexToRemove], c.Resources[indexToRemove+1:]...)
	return nil
}

// SetResourceWebhooks implements config.Config
func (c *Cfg) SetResourceWebhooks(gvk resource.GVK, webhooks *resource.Webhooks) error {
	for i, r := range c.Resources {
		// Match by Group, Version, Kind (not domain)
		if r.Group == gvk.Group && r.Version == gvk.Version && r.Kind == gvk.Kind {
			if webhooks == nil {
				c.Resources[i].Webhooks = nil
			} else {
				if c.Resources[i].Webhooks == nil {
					c.Resources[i].Webhooks = &resource.Webhooks{}
				}
				c.Resources[i].Webhooks.Set(webhooks)
			}
			return nil
		}
	}

	return fmt.Errorf("failed to set webhooks: resource with GVK {%q %q %q} not found",
		gvk.Group, gvk.Version, gvk.Kind)
}

// HasGroup implements config.Config
func (c Cfg) HasGroup(group string) bool {
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
func (c Cfg) ListCRDVersions() []string {
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
func (c Cfg) ListWebhookVersions() []string {
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
func (c Cfg) DecodePluginConfig(key string, configObj any) error {
	if len(c.Plugins) == 0 {
		return config.PluginKeyNotFoundError{Key: key}
	}

	// Get the object blob by key and unmarshal into the object.
	if pluginCfg, hasKey := c.Plugins[key]; hasKey {
		b, err := yaml.Marshal(pluginCfg)
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
//
// Plugin Configuration Deletion:
//
// To remove a plugin's configuration from the PROJECT file, pass an anonymous empty struct (struct{}{}).
// This explicitly signals the intent to delete the plugin's configuration entry.
//
// Example:
//
//	// Delete plugin configuration
//	err := config.EncodePluginConfig("my-plugin.example.com/v1", struct{}{})
//
// Note: Named empty structs or structs with omitempty fields that evaluate to empty
// will be stored as "{}" in the PROJECT file, not deleted. Only struct{}{} triggers deletion.
func (c *Cfg) EncodePluginConfig(key string, configObj any) error {
	// Get object's bytes and set them under key in extra fields.
	b, err := yaml.Marshal(configObj)
	if err != nil {
		return fmt.Errorf("failed to convert %T object to bytes: %w", configObj, err)
	}
	var fields map[string]any
	if err := yaml.Unmarshal(b, &fields); err != nil {
		return fmt.Errorf("failed to unmarshal %T object bytes: %w", configObj, err)
	}
	if c.Plugins == nil {
		c.Plugins = make(map[string]pluginConfig)
	}
	// If fields is empty and configObj is an anonymous empty struct,
	// delete the key from the plugins map (used for plugin deletion).
	// Named structs with omitempty fields that evaluate to empty will still be stored as {}.
	if len(fields) == 0 && isAnonymousEmptyStruct(configObj) {
		delete(c.Plugins, key)
	} else {
		c.Plugins[key] = fields
	}
	return nil
}

// isAnonymousEmptyStruct checks if the given value is an anonymous empty struct (struct{}{}).
//
// This function is used by EncodePluginConfig to distinguish between:
//   - struct{}{} → explicit deletion signal (returns true)
//   - type MyConfig struct{} → empty named struct (returns false, will be stored as {})
//   - Structs with omitempty fields that evaluate to empty (returns false, will be stored as {})
//
// Only anonymous empty structs (struct{}{}) trigger configuration deletion from the PROJECT file.
func isAnonymousEmptyStruct(v any) bool {
	typ := reflect.TypeOf(v)
	return typ.Kind() == reflect.Struct && typ.Name() == "" && typ.NumField() == 0
}

// MarshalYAML implements config.Config
func (c Cfg) MarshalYAML() ([]byte, error) {
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

// UnmarshalYAML implements config.Config
func (c *Cfg) UnmarshalYAML(b []byte) error {
	if err := yaml.UnmarshalStrict(b, c); err != nil {
		return config.UnmarshalError{Err: err}
	}

	return nil
}
