.PHONY: webhook-certs
webhook-certs: ## Generate and inject development webhook certificates
	./hack/webhook-certs.sh my-project
