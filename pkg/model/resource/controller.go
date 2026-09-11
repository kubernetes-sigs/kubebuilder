/*
Copyright 2026 The Kubernetes Authors.

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

package resource

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

// Controller represents a named controller for a resource.
// Each controller has a unique name that identifies it within a resource (GVK).
type Controller struct {
	// Name is the controller identifier, unique within a resource.
	// Must be a valid DNS label (lowercase, alphanumeric, hyphens, max 63 chars).
	Name string `json:"name"`
}

// Validate checks that the Controller is valid.
func (c Controller) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("controller name cannot be empty")
	}

	// Controller names must be valid DNS labels
	if errors := validation.IsDNS1035Label(c.Name); len(errors) != 0 {
		return fmt.Errorf("invalid controller name %q: %s", c.Name, strings.Join(errors, ", "))
	}

	return nil
}

// Controllers holds a list of controllers for a resource.
type Controllers []Controller

// IsEmpty returns true if there are no controllers.
func (c *Controllers) IsEmpty() bool {
	return c == nil || len(*c) == 0
}

// Validate checks that all controllers are valid and have unique names.
// Names generating the same reconciler struct are rejected by Resource.Validate,
// which knows the kind those names are resolved against.
func (c *Controllers) Validate() error {
	if c.IsEmpty() {
		return nil
	}

	names := make(map[string]bool)

	for _, controller := range *c {
		if err := controller.Validate(); err != nil {
			return err
		}

		if names[controller.Name] {
			return fmt.Errorf("duplicate controller name %q", controller.Name)
		}
		names[controller.Name] = true
	}

	return nil
}

// DefaultControllerName returns the controller name used when none is given.
func DefaultControllerName(kind string) string {
	return strings.ToLower(kind)
}

// NormalizeFileName converts a controller name to a valid Go filename.
// Hyphens are replaced with underscores to follow Go file naming conventions.
// Example: "captain-backup" becomes "captain_backup".
func NormalizeFileName(controllerName string) string {
	return strings.ReplaceAll(controllerName, "-", "_")
}

// NormalizeReconcilerName converts a controller name to a PascalCase reconciler struct name.
// For backward compatibility, returns "{Kind}Reconciler" when the controller name
// is empty or matches the lowercase kind.
// Example: "captain-backup" becomes "CaptainBackupReconciler".
func NormalizeReconcilerName(controllerName, kind string) string {
	// Backward compatible: no controller name or name matches kind
	if controllerName == "" || controllerName == strings.ToLower(kind) {
		return kind + "Reconciler"
	}

	// Convert hyphenated name to PascalCase (e.g., "captain-backup" -> "CaptainBackup")
	parts := strings.Split(controllerName, "-")
	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]))
			result.WriteString(part[1:])
		}
	}
	result.WriteString("Reconciler")
	return result.String()
}

// GetControllerName returns the runtime name used in Named() and error logs.
// In multigroup projects, the group name is prefixed to avoid naming conflicts.
// Examples: "captain" in single-group, "crew-captain" in multigroup.
func GetControllerName(controllerName, kind, group string, multiGroup bool) string {
	var name string
	if controllerName != "" {
		name = controllerName
	} else {
		name = strings.ToLower(kind)
	}

	// In multigroup mode, prefix with the group name
	if multiGroup && group != "" {
		return strings.ToLower(group) + "-" + name
	}

	return name
}

// HasController returns true if a controller with the given name exists.
func (c *Controllers) HasController(name string) bool {
	if c.IsEmpty() {
		return false
	}

	for _, controller := range *c {
		if controller.Name == name {
			return true
		}
	}
	return false
}

// AddController adds a new controller with the given name.
// Returns an error if a controller with that name already exists.
func (c *Controllers) AddController(name string) error {
	if c == nil {
		return fmt.Errorf("cannot add controller to nil Controllers")
	}

	controller := Controller{Name: name}
	if err := controller.Validate(); err != nil {
		return err
	}

	if c.HasController(name) {
		return fmt.Errorf("controller with name %q already exists", name)
	}

	*c = append(*c, controller)
	return nil
}

// GetControllerNames returns a slice of all controller names.
func (c *Controllers) GetControllerNames() []string {
	if c.IsEmpty() {
		return nil
	}

	names := make([]string, 0, len(*c))
	for _, controller := range *c {
		names = append(names, controller.Name)
	}
	return names
}

// Copy returns a deep copy of the Controllers.
func (c *Controllers) Copy() Controllers {
	if c == nil {
		return Controllers{}
	}

	controllers := make(Controllers, len(*c))
	copy(controllers, *c)
	return controllers
}

// Update combines fields of two Controllers.
// It adds controllers from other that don't exist in c.
func (c *Controllers) Update(other *Controllers) error {
	if c == nil {
		return fmt.Errorf("cannot update a nil Controllers")
	}

	if other == nil || other.IsEmpty() {
		return nil
	}

	for _, controller := range *other {
		if !c.HasController(controller.Name) {
			*c = append(*c, controller)
		}
	}

	return nil
}
