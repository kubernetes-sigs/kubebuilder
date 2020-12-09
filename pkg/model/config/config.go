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

	"sigs.k8s.io/kubebuilder/v2/pkg/model/resource"
)

// Scaffolding versions
const (
	Version2      = "2"
	Version3Alpha = "3-alpha"
)

// Config is the unmarshalled representation of the configuration file
// NOTE: when adding new fields, add them to validate to ensure that
//   they are only found in the corresponding versions.
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
	Resources []*resource.Resource `json:"resources,omitempty"`

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

type unsupportedFieldError struct {
	fieldName string
	version   string
}

func (e unsupportedFieldError) Error() string {
	return fmt.Sprintf("`%s` field found for %s", e.fieldName, e.version)
}

func (c Config) errorForField(fieldName string) error {
	return unsupportedFieldError{fieldName: fieldName, version: c.Version}
}

// validate checks if a Config is valid or it has some fields that shouldn't be present for that version
func (c Config) validate() error {
	if c.IsV2() {
		if c.ProjectName != "" {
			return c.errorForField("projectName")
		}
		for _, r := range c.Resources {
			if r.Domain != "" {
				return c.errorForField("resources[].domain")
			}
			if r.Plural != "" {
				return c.errorForField("resources[].plural")
			}
			if r.Path != "" {
				return c.errorForField("resources[].plural")
			}
			if r.API != nil {
				return c.errorForField("resources[].api")
			}
			if r.Controller {
				return c.errorForField("resources[].controller")
			}
			if r.Webhooks != nil {
				return c.errorForField("resources[].webhooks")
			}
		}
		if c.ComponentConfig {
			return c.errorForField("componentConfig")
		}
		if c.Layout != "" {
			return c.errorForField("layout")
		}
		if len(c.Plugins) != 0 {
			return c.errorForField("plugins")
		}
	}

	return nil
}

// GetResource returns the requested resource if it is already tracked
func (c Config) GetResource(target resource.GVK) *resource.Resource {
	// Return the target resource if it is found in the tracked resources
	for _, r := range c.Resources {
		if r.GVK().IsEqualTo(target) {
			return r
		}
	}

	// Return nil otherwise
	return nil
}

// UpdateResources adds the resource to the tracked ones or updates it as needed
func (c *Config) UpdateResources(res *resource.Resource) (*resource.Resource, error) {
	// Create a copy of the resource to ensure we do not accidentally modify it
	resCopy := *res

	// V2 only requires group, version and kind fields
	if c.IsV2() {
		resCopy = resource.Resource{
			Group:   resCopy.Group,
			Version: resCopy.Version,
			Kind:    resCopy.Kind,
		}
	}

	gvk := resCopy.GVK()
	// If the resource already exists, update it.
	for i, r := range c.Resources {
		if r.GVK().IsEqualTo(gvk) {
			err := c.Resources[i].Update(&resCopy)
			return c.Resources[i], err
		}
	}

	// The resource does not exist, append the resource to the tracked ones.
	c.Resources = append(c.Resources, &resCopy)
	return res, nil
}

// HasGroup returns true if group is already tracked
func (c Config) HasGroup(group string) bool {
	// Return true if the target group is found in the tracked resources
	for _, r := range c.Resources {
		if strings.EqualFold(group, r.QualifiedGroup()) {
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
				currVersion = res.API.Version
			}
		case "webhook":
			if res.Webhooks != nil {
				currVersion = res.Webhooks.Version
			}
		}
		if currVersion != "" && version != currVersion {
			return false
		}
	}
	return true
}

// simplify omits some fields that can be restored at unmarshalling.
func (c *Config) simplify() {
	if c.IsV3() {
		for i, r := range c.Resources {
			// If the plural is regular, omit it.
			if r.Plural == resource.RegularPlural(r.Kind) {
				c.Resources[i].Plural = ""
			}
			// If the path is the default location, omit it.
			if r.Path == resource.LocalPath(c.Repo, r.Group, r.Version, c.MultiGroup) {
				c.Resources[i].Path = ""
			}
			// If API is empty, omit it (prevents `api: {}`).
			if r.API != nil && r.API.IsEmpty() {
				c.Resources[i].API = nil
			}
			// If Webhooks is empty, omit it (prevents `webhooks: {}`).
			if r.Webhooks != nil && r.Webhooks.IsEmpty() {
				c.Resources[i].Webhooks = nil
			}
		}
	}
}

// restore sets the omitted values that were simplified during marshalling.
func (c *Config) restore() {
	if c.IsV3() {
		for i, r := range c.Resources {
			// Restore regular plural that were omitted.
			if r.Plural == "" {
				c.Resources[i].Plural = resource.RegularPlural(r.Kind)
			}
			// Restore the default location that was omitted.
			if r.Path == "" {
				c.Resources[i].Path = resource.LocalPath(c.Repo, r.Group, r.Version, c.MultiGroup)
			}
			// Create a pointer to an empty struct instead of nil to prevent panics.
			if r.API == nil {
				c.Resources[i].API = &resource.API{}
			}
			// Create a pointer to an empty struct instead of nil to prevent panics.
			if r.Webhooks == nil {
				c.Resources[i].Webhooks = &resource.Webhooks{}
			}
		}
	}
}

type marshalError struct {
	error
}

func (e marshalError) Error() string {
	return fmt.Sprintf("error marshalling project configuration: %s", e.error.Error())
}

func (e marshalError) Unwrap() error {
	return e.error
}

// Marshal returns the bytes of c.
func (c Config) Marshal() ([]byte, error) {
	// Validate to verify that no unsupported field is found
	if err := c.validate(); err != nil {
		return nil, marshalError{err}
	}

	// Make a copy of the config in order not to mess with the real configuration.
	cfg := c

	// Omit those fields that we can restore at unmarshalling to reduce the size of the project configuration file
	cfg.simplify()

	// Ignore extra fields at first when marshalling.
	cfg.Plugins = nil
	content, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, marshalError{err}
	}

	// Empty config strings are "{}" due to the map field.
	if strings.TrimSpace(string(content)) == "{}" {
		content = []byte{}
	}

	// Append extra fields to put them at the config's bottom.
	if len(c.Plugins) != 0 {
		pluginConfigBytes, err := yaml.Marshal(Config{Plugins: c.Plugins})
		if err != nil {
			return nil, marshalError{err}
		}

		content = append(content, pluginConfigBytes...)
	}

	return content, nil
}

type unmarshalError struct {
	error
}

func (e unmarshalError) Error() string {
	return fmt.Sprintf("error unmarshalling project configuration: %s", e.error.Error())
}

func (e unmarshalError) Unwrap() error {
	return e.error
}

// Unmarshal unmarshalls the bytes of a Config into c.
func (c *Config) Unmarshal(b []byte) error {
	if err := yaml.UnmarshalStrict(b, c); err != nil {
		return unmarshalError{err}
	}

	// To simplify the project configuration files we have omitted some fields that could be restored, do it now
	c.restore()

	// Validate to verify that no unsupported field is found
	if err := c.validate(); err != nil {
		return unmarshalError{err}
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
		c.Plugins = make(PluginConfigs)
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
