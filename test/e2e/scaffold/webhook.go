package scaffold

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

// Controller scaffolds a Controller for a Resource
type Webhook struct {
	Domain         string
	Group          string
	DomainWithDash string
	Version        string
	Kind           string
	Resources      string
}

func (w *Webhook) validate() error {
	if len(w.Domain) == 0 {
		return fmt.Errorf("domain should not be empty")
	}
	if len(w.Group) == 0 {
		return fmt.Errorf("group should not be empty")
	}
	if len(w.Version) == 0 {
		return fmt.Errorf("version should not be empty")
	}
	if len(w.Kind) == 0 {
		return fmt.Errorf("kind should not be empty")
	}
	if len(w.Resources) == 0 {
		return fmt.Errorf("resoureces should not be empty")
	}
	return nil
}

// GetInput implements input.File
func (w *Webhook) WriteTo(filename string) error {
	if err := w.validate(); err != nil {
		return err
	}

	w.DomainWithDash = strings.Replace(w.Domain, ".", "-", -1)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	t := template.Must(template.New("Webhook").
		Funcs(template.FuncMap{"lower": strings.ToLower}).
		Parse(webhookTemplate))
	return t.Execute(f, w)
}

var webhookTemplate = `package {{ .Version }}

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// +kubebuilder:webhook:failurePolicy=fail,groups={{ .Group }}.{{ .Domain }},resources={{ .Resources }},verbs=create;update,versions={{ .Version }},name=v{{ lower .Kind }}.{{ .Domain }},path=/validate-{{ .Group }}-{{ .DomainWithDash }}-{{ .Version }}-{{ lower .Kind }},mutating=false

var _ webhook.Validator = &{{ .Kind }}{}

func (c *{{ .Kind }}) ValidateCreate() error {
	if c.Spec.Count < 0 {
		return fmt.Errorf(".spec.count must >= 0")
	}
	return nil
}

func (c *{{ .Kind }}) ValidateUpdate(old runtime.Object) error {
	if c.Spec.Count < 0 {
		return fmt.Errorf(".spec.count must >= 0")
	}
	return nil
}

// +kubebuilder:webhook:failurePolicy=fail,groups={{ .Group }}.{{ .Domain }},resources={{ .Resources }},verbs=create;update,versions={{ .Version }},name=m{{ lower .Kind }}.{{ .Domain }},path=/mutate-{{ .Group }}-{{ .DomainWithDash }}-{{ .Version }}-{{ lower .Kind }},mutating=true

var _ webhook.Defaulter = &{{ .Kind }}{}

func (c *{{ .Kind }}) Default() {
	if c.Spec.Count == 0 {
		c.Spec.Count = 5
	}
}
`
