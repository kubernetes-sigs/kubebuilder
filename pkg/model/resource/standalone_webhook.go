package resource

import "fmt"

// Webhook defines a webhook that intercepts multiple resource types
// (multiple GVKs) rather than being tied to a single API resource.
type Webhook struct {
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

	// Kinds is the list of resource kinds the webhook intercepts (e.g., pods, deployments).
	Kinds []string `json:"resources,omitempty"`

	// Versions is the list of API versions the webhook intercepts, or "*" for all.
	Versions []string `json:"versions,omitempty"`

	// DefaultingPath holds the custom path for the defaulting/mutating webhook.
	DefaultingPath string `json:"defaultingPath,omitempty"`

	// ValidationPath holds the custom path for the validation webhook.
	ValidationPath string `json:"validationPath,omitempty"`
}

// Validate checks that the Webhook is structurally valid.
// Business-logic checks (e.g., requiring at least one group/resource/version)
// are handled at the plugin level.
func (w Webhook) Validate() error {
	if w.WebhookVersion != "" {
		if err := validateAPIVersion(w.WebhookVersion); err != nil {
			return fmt.Errorf("invalid webhook version for %q: %w", w.Name, err)
		}
	}
	return nil
}

// IsEmpty returns true if the Webhook has no meaningful configuration.
func (w Webhook) IsEmpty() bool {
	return w.Name == "" && !w.Defaulting && !w.Validation &&
		len(w.Groups) == 0 && len(w.Kinds) == 0 && len(w.Versions) == 0
}

// Copy returns a deep copy of the Webhook.
func (w Webhook) Copy() Webhook {
	groups := make([]string, len(w.Groups))
	copy(groups, w.Groups)
	kinds := make([]string, len(w.Kinds))
	copy(kinds, w.Kinds)
	versions := make([]string, len(w.Versions))
	copy(versions, w.Versions)
	return Webhook{
		Name:           w.Name,
		WebhookVersion: w.WebhookVersion,
		Defaulting:     w.Defaulting,
		Validation:     w.Validation,
		Groups:         groups,
		Kinds:          kinds,
		Versions:       versions,
		DefaultingPath: w.DefaultingPath,
		ValidationPath: w.ValidationPath,
	}
}
