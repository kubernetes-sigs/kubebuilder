package resource

import "fmt"

// StandaloneWebhook defines a webhook that intercepts multiple resource types
// (multiple GVKs) rather than being tied to a single API resource.
type StandaloneWebhook struct {
	// Name is a unique identifier for this webhook (used in the handler name, path, and PROJECT entry).
	Name string `json:"name,omitempty"`

	// WebhookVersion holds the {Validating,Mutating}WebhookConfiguration API version.
	WebhookVersion string `json:"webhookVersion,omitempty"`

	// Defaulting specifies if a defaulting/mutating webhook is scaffolded.
	Defaulting bool `json:"defaulting,omitempty"`

	// Validation specifies if a validating webhook is scaffolded.
	Validation bool `json:"validation,omitempty"`

	// Groups is the list of API groups the webhook intercepts. Use "" for the core group.
	Groups []string `json:"groups,omitempty"`

	// Resources is the list of resource types the webhook intercepts (e.g., pods, deployments).
	Resources []string `json:"resources,omitempty"`

	// Versions is the list of API versions the webhook intercepts, or "*" for all.
	Versions []string `json:"versions,omitempty"`

	// DefaultingPath holds the custom path for the defaulting/mutating webhook.
	DefaultingPath string `json:"defaultingPath,omitempty"`

	// ValidationPath holds the custom path for the validation webhook.
	ValidationPath string `json:"validationPath,omitempty"`
}

// Validate checks that the StandaloneWebhook is valid.
func (w StandaloneWebhook) Validate() error {
	if w.Name == "" {
		return fmt.Errorf("standalone webhook name cannot be empty")
	}
	if !w.Defaulting && !w.Validation {
		return fmt.Errorf("standalone webhook %q requires at least one of defaulting or validation", w.Name)
	}
	if len(w.Groups) == 0 {
		return fmt.Errorf("standalone webhook %q requires at least one group", w.Name)
	}
	if len(w.Resources) == 0 {
		return fmt.Errorf("standalone webhook %q requires at least one resource", w.Name)
	}
	if len(w.Versions) == 0 {
		return fmt.Errorf("standalone webhook %q requires at least one version", w.Name)
	}
	if w.WebhookVersion != "" {
		if err := validateAPIVersion(w.WebhookVersion); err != nil {
			return fmt.Errorf("invalid webhook version for %q: %w", w.Name, err)
		}
	}
	return nil
}

// IsEmpty returns true if the StandaloneWebhook has no meaningful configuration.
func (w StandaloneWebhook) IsEmpty() bool {
	return w.Name == "" && !w.Defaulting && !w.Validation &&
		len(w.Groups) == 0 && len(w.Resources) == 0 && len(w.Versions) == 0
}

// Copy returns a deep copy of the StandaloneWebhook.
func (w StandaloneWebhook) Copy() StandaloneWebhook {
	groups := make([]string, len(w.Groups))
	copy(groups, w.Groups)
	resources := make([]string, len(w.Resources))
	copy(resources, w.Resources)
	versions := make([]string, len(w.Versions))
	copy(versions, w.Versions)
	return StandaloneWebhook{
		Name:           w.Name,
		WebhookVersion: w.WebhookVersion,
		Defaulting:     w.Defaulting,
		Validation:     w.Validation,
		Groups:         groups,
		Resources:      resources,
		Versions:       versions,
		DefaultingPath: w.DefaultingPath,
		ValidationPath: w.ValidationPath,
	}
}
