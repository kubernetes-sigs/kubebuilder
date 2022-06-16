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

//go:deprecated This package has been deprecated
package v2

import (
	"strings"

	"sigs.k8s.io/yaml"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

// Version is the config.Version for project configuration 2
var Version = config.Version{Number: 2}

type cfg struct {
	// Version
	Version config.Version `json:"version"`

	// String fields
	Domain     string `json:"domain,omitempty"`
	Repository string `json:"repo,omitempty"`

	// Boolean fields
	MultiGroup bool `json:"multigroup,omitempty"`

	// Resources
	Gvks []resource.GVK `json:"resources,omitempty"`
}

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
	return ""
}

// SetProjectName implements config.Config
func (c *cfg) SetProjectName(string) error {
	return config.UnsupportedFieldError{
		Version: Version,
		Field:   "project name",
	}
}

// GetPluginChain implements config.Config
func (c cfg) GetPluginChain() []string {
	return []string{"go.kubebuilder.io/v2"}
}

// SetPluginChain implements config.Config
func (c *cfg) SetPluginChain([]string) error {
	return config.UnsupportedFieldError{
		Version: Version,
		Field:   "plugin chain",
	}
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
	return false
}

// SetComponentConfig implements config.Config
func (c *cfg) SetComponentConfig() error {
	return config.UnsupportedFieldError{
		Version: Version,
		Field:   "component config",
	}
}

// ClearComponentConfig implements config.Config
func (c *cfg) ClearComponentConfig() error {
	return config.UnsupportedFieldError{
		Version: Version,
		Field:   "component config",
	}
}

// ResourcesLength implements config.Config
func (c cfg) ResourcesLength() int {
	return len(c.Gvks)
}

// HasResource implements config.Config
func (c cfg) HasResource(gvk resource.GVK) bool {
	gvk.Domain = "" // Version 2 does not include domain per resource

	for _, trackedGVK := range c.Gvks {
		if gvk.IsEqualTo(trackedGVK) {
			return true
		}
	}

	return false
}

// GetResource implements config.Config
func (c cfg) GetResource(gvk resource.GVK) (resource.Resource, error) {
	gvk.Domain = "" // Version 2 does not include domain per resource

	for _, trackedGVK := range c.Gvks {
		if gvk.IsEqualTo(trackedGVK) {
			return resource.Resource{
				GVK: trackedGVK,
			}, nil
		}
	}

	return resource.Resource{}, config.ResourceNotFoundError{GVK: gvk}
}

// GetResources implements config.Config
func (c cfg) GetResources() ([]resource.Resource, error) {
	resources := make([]resource.Resource, 0, len(c.Gvks))
	for _, gvk := range c.Gvks {
		resources = append(resources, resource.Resource{
			GVK: gvk,
		})
	}

	return resources, nil
}

// AddResource implements config.Config
func (c *cfg) AddResource(res resource.Resource) error {
	// As res is passed by value it is already a shallow copy, and we are only using
	// fields that do not require a deep copy, so no need to make a deep copy

	res.Domain = "" // Version 2 does not include domain per resource

	if !c.HasResource(res.GVK) {
		c.Gvks = append(c.Gvks, res.GVK)
	}

	return nil
}

// UpdateResource implements config.Config
func (c *cfg) UpdateResource(res resource.Resource) error {
	return c.AddResource(res)
}

// HasGroup implements config.Config
func (c cfg) HasGroup(group string) bool {
	// Return true if the target group is found in the tracked resources
	for _, r := range c.Gvks {
		if strings.EqualFold(group, r.Group) {
			return true
		}
	}

	// Return false otherwise
	return false
}

// ListCRDVersions implements config.Config
func (c cfg) ListCRDVersions() []string {
	return make([]string, 0)
}

// ListWebhookVersions implements config.Config
func (c cfg) ListWebhookVersions() []string {
	return make([]string, 0)
}

// DecodePluginConfig implements config.Config
func (c cfg) DecodePluginConfig(string, interface{}) error {
	return config.UnsupportedFieldError{
		Version: Version,
		Field:   "plugins",
	}
}

// EncodePluginConfig implements config.Config
func (c cfg) EncodePluginConfig(string, interface{}) error {
	return config.UnsupportedFieldError{
		Version: Version,
		Field:   "plugins",
	}
}

// Marshal implements config.Config
func (c cfg) MarshalYAML() ([]byte, error) {
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
