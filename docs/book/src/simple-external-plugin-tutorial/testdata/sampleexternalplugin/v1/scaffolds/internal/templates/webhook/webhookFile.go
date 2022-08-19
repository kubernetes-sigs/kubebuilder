package webhook

// WebhookFile represents the WebhookFile.txt
type WebhookFile struct {
	Name     string
	Contents string
	hooked   bool
}

// WebhookFileOptions is a way to set configurable options for the Webhook file
type WebhookFileOptions func(wf *WebhookFile)

// WithHooked sets whether or not to add `HOOKED` to a new line in the resulting WebhookFile
func WithHooked(hooked bool) WebhookFileOptions {
	return func(wf *WebhookFile) {
		wf.hooked = hooked
	}
}

// NewWebhookFile returns a new WebhookFile with
func NewWebhookFile(opts ...WebhookFileOptions) *WebhookFile {
	webhookFile := &WebhookFile{
		Name: "webhookFile.txt",
	}

	for _, opt := range opts {
		opt(webhookFile)
	}

	webhookFile.Contents = WebhookFileDefaultMessage

	if webhookFile.hooked {
		webhookFile.Contents += WebhookFileHookedMessage
	}

	return webhookFile
}

const WebhookFileDefaultMessage = "A simple text file created with the `create webhook` subcommand"
const WebhookFileHookedMessage = "\nHOOKED!"
