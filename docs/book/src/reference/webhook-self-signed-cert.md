# Webhook with Self-Signed Certificates (Without cert-manager)

<aside class="warning" role="note">
<p class="note-title">Warning</p>
This guide shows how to configure webhook TLS using manually generated
self-signed certificates. This approach is intended **for development and
testing environments only** (e.g. local `kind` clusters). It is **not
recommended for production**.

For production, use [cert-manager](https://cert-manager.io) or another certificate
manager of your choice, as described in
[Deploying cert-manager](../cronjob-tutorial/cert-manager.md).
</aside>

## Overview

By default, Kubebuilder scaffolds webhook configuration that relies on
[cert-manager](https://cert-manager.io) to provision and inject TLS
certificates. If you want to run webhooks in a development cluster **without**
installing cert-manager, you can automate certificate generation and injection using a simple helper script and a Makefile target.

The script will:
1. Generate a self-signed CA and TLS certificate with `openssl`.
2. Create the `webhook-server-cert` Secret in your cluster.
3. Automatically patch the CA bundle into your `MutatingWebhookConfiguration`, `ValidatingWebhookConfiguration`, and CRD conversion webhooks.

---

## Prerequisites

- A Kubebuilder project with at least one webhook scaffolded.
- `openssl` available on your machine.
- A running cluster (e.g. [kind](./kind.md)).
- `kubectl` configured to talk to the cluster.

---

## Step 1: Create the helper script

Create a file named `hack/webhook-certs.sh` in your project and add the following content:

```bash
#!/usr/bin/env bash

set -e

# Usage: ./hack/webhook-certs.sh <project-name>

PROJECT_NAME=${1:-"my-project"}
NAMESPACE="${PROJECT_NAME}-system"
SERVICE="${PROJECT_NAME}-webhook-service"
CERT_DIR="config/webhook/certs"

echo "Generating certificates for ${SERVICE}.${NAMESPACE}.svc..."
mkdir -p "${CERT_DIR}"

# 1. Generate the CA private key and self-signed certificate
openssl genrsa -out "${CERT_DIR}/ca.key" 2048
openssl req -new -x509 -days 365 \
  -key "${CERT_DIR}/ca.key" \
  -subj "/CN=Webhook CA/O=Dev" \
  -out "${CERT_DIR}/ca.crt"

# 2. Generate the webhook server private key
openssl genrsa -out "${CERT_DIR}/tls.key" 2048

# 3. Generate a CSR for the webhook service DNS name
openssl req -new \
  -key "${CERT_DIR}/tls.key" \
  -subj "/CN=${SERVICE}.${NAMESPACE}.svc" \
  -out "${CERT_DIR}/tls.csr"

# 4. Create a SAN extension file
cat > "${CERT_DIR}/san.ext" <<EOF
subjectAltName = DNS:${SERVICE}.${NAMESPACE}.svc, DNS:${SERVICE}.${NAMESPACE}.svc.cluster.local
EOF

# 5. Sign the server certificate with the CA
openssl x509 -req -days 365 \
  -in "${CERT_DIR}/tls.csr" \
  -CA "${CERT_DIR}/ca.crt" \
  -CAkey "${CERT_DIR}/ca.key" \
  -CAcreateserial \
  -extfile "${CERT_DIR}/san.ext" \
  -out "${CERT_DIR}/tls.crt"

# Clean up temporary CSR and SAN extension files
rm "${CERT_DIR}/tls.csr" "${CERT_DIR}/san.ext"

# Base64 encode the CA bundle
CA_BUNDLE=$(base64 -w 0 < "${CERT_DIR}/ca.crt")

echo "Creating or updating the webhook-server-cert Secret in namespace ${NAMESPACE}..."
kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
kubectl create secret tls webhook-server-cert \
  --cert="${CERT_DIR}/tls.crt" \
  --key="${CERT_DIR}/tls.key" \
  --namespace="${NAMESPACE}" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "Injecting CA bundle into MutatingWebhookConfigurations..."
for webhook in $(kubectl get mutatingwebhookconfiguration -o name | grep "${PROJECT_NAME}" || true); do
    WEBHOOKS_COUNT=$(kubectl get "$webhook" -o jsonpath='{range .webhooks[*]}{@.name}{"\n"}{end}' | wc -l)
    for (( i=0; i<WEBHOOKS_COUNT; i++ )); do
        kubectl patch "$webhook" --type='json' -p="[{\"op\": \"replace\", \"path\": \"/webhooks/$i/clientConfig/caBundle\", \"value\": \"${CA_BUNDLE}\"}]"
    done
done

echo "Injecting CA bundle into ValidatingWebhookConfigurations..."
for webhook in $(kubectl get validatingwebhookconfiguration -o name | grep "${PROJECT_NAME}" || true); do
    WEBHOOKS_COUNT=$(kubectl get "$webhook" -o jsonpath='{range .webhooks[*]}{@.name}{"\n"}{end}' | wc -l)
    for (( i=0; i<WEBHOOKS_COUNT; i++ )); do
        kubectl patch "$webhook" --type='json' -p="[{\"op\": \"replace\", \"path\": \"/webhooks/$i/clientConfig/caBundle\", \"value\": \"${CA_BUNDLE}\"}]"
    done
done

echo "Injecting CA bundle into CRD conversion webhooks..."
for crd in $(kubectl get crd -o name | grep "${PROJECT_NAME}" || true); do
    HAS_WEBHOOK=$(kubectl get "$crd" -o jsonpath='{.spec.conversion.webhook.clientConfig}' || true)
    if [ -n "$HAS_WEBHOOK" ]; then
        kubectl patch "$crd" --type='json' -p="[{\"op\": \"replace\", \"path\": \"/spec/conversion/webhook/clientConfig/caBundle\", \"value\": \"${CA_BUNDLE}\"}]"
    fi
done

echo "Done!"
```

Make the script executable:

```bash
chmod +x hack/webhook-certs.sh
```

Ensure the generated certificates aren't accidentally tracked by Git:

```bash
echo "config/webhook/certs/" >> .gitignore
```

---

## Step 2: Add a Makefile target

Open your `Makefile` and add the following target to automate running the script:

```makefile
.PHONY: webhook-certs
webhook-certs: ## Generate and inject development webhook certificates
	./hack/webhook-certs.sh my-project
```

*(Note: Make sure to replace `my-project` with the actual name prefix of your project)*

---

## Step 3: Disable cert-manager in Kustomize

Open `config/default/kustomization.yaml` and make the following changes:

**3a. Remove (or comment out) the `../certmanager` resource:**

```yaml
resources:
- ../crd
- ../rbac
- ../manager
- ../webhook
# [CERTMANAGER] Comment out the line below to disable cert-manager
#- ../certmanager
```

**3b. Remove (or comment out) all `replacements` blocks** that reference
`cert-manager.io` resources. These are the blocks that inject
`cert-manager.io/inject-ca-from` annotations into your webhook configurations.
They all start with a `- source:` entry whose `kind` is `Certificate` or whose
`targets` select a `kind: Certificate`. Comment out the entire `replacements:`
section if all entries are cert-manager related.

After editing, the bottom of your `kustomization.yaml` should look similar to:

```yaml
patches:
- path: manager_metrics_patch.yaml
  target:
    kind: Deployment
- path: manager_webhook_patch.yaml
  target:
    kind: Deployment

# replacements: # all commented out — cert-manager not used
```

---

## Step 4: Build, Deploy and Inject

Now, you can build and deploy your project, followed by injecting the certificates:

```bash
# Build and push your controller image
make docker-build docker-push IMG=<your-registry>/my-project:dev

# Deploy (cert-manager is not needed)
make deploy IMG=<your-registry>/my-project:dev

# Generate certificates and inject them into the deployed webhooks
make webhook-certs
```

Verify the webhook pod is running and the certificate secret was mounted:

```bash
kubectl get pods -n <your-project>-system
kubectl describe secret webhook-server-cert -n <your-project>-system
```

---

## Alternative: `make run` (local process, not in-cluster)

When running the controller locally with `make run`, the webhook server needs
TLS certificates on the local filesystem.

First, run the script to generate the certificates in the `config/webhook/certs` directory:

```bash
make webhook-certs
```

Then, point controller-runtime to the certificates using the `--webhook-cert-path` flag:

```bash
make run ARGS="--webhook-cert-path=config/webhook/certs"
```

You still need to:
- Forward or expose the webhook service so the API server can reach it, **or**
- Disable webhooks locally (e.g. set `failurePolicy: Ignore` temporarily).

For a smoother local development loop without any certificates, consider running
the API server locally with webhooks disabled or using
[envtest](./envtest.md) which lets you test webhook logic without a real cluster.

---

## References

- [cert-manager (recommended for production)](https://cert-manager.io)
- [Deploying cert-manager](../cronjob-tutorial/cert-manager.md)
- [Deploying webhooks](../cronjob-tutorial/running-webhook.md)
- [Kind for Dev and CI](./kind.md)
