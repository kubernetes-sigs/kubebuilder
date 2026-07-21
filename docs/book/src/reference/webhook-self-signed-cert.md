# Webhook with Self-Signed Certificates (Without cert-manager)

<aside class="warning" role="note">

<p class="note-title">Warning</p>

This guide shows how to configure webhook TLS using manually generated
self-signed certificates. This approach is intended **for development and
testing environments only** (e.g. local `kind` clusters). It is **not
recommended for production** because:

- Certificates must be rotated manually.
- The CA bundle must be patched into the webhook configuration by hand.
- There is no automated renewal or revocation.

For production, use [cert-manager](https://cert-manager.io) or another certificate manager of your choice, as described in
[Deploying cert-manager](../cronjob-tutorial/cert-manager.md).

</aside>

## Overview

By default, Kubebuilder scaffolds webhook configuration that relies on
[cert-manager](https://cert-manager.io) to provision and inject TLS
certificates. If you want to run webhooks in a development cluster **without**
installing cert-manager, you can:

1. Generate a self-signed CA and TLS certificate with `openssl`.
2. Create a Kubernetes `Secret` from those certificates.
3. Disable the cert-manager components in your Kustomize configuration.
4. Patch the `caBundle` field in the webhook configuration manually.

---

## Prerequisites

- A Kubebuilder project with at least one webhook scaffolded.
- `openssl` available on your machine.
- A running cluster (e.g. [kind](./kind.md)).
- `kubectl` configured to talk to the cluster.

---

## Step 1: Generate self-signed certificates

To simplify certificate generation, you can use the following script. Save it to `hack/generate-certs.sh` and make it executable:

```bash
#!/usr/bin/env bash

# Set the namespace and service name
NAMESPACE=${1:-"my-project-system"}
SERVICE=${2:-"webhook-service"}
CERT_DIR="config/webhook/certs"

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

# Clean up CSR and SAN extension files
rm "${CERT_DIR}/tls.csr" "${CERT_DIR}/san.ext"
```

Run the script by specifying your webhook namespace:

```bash
chmod +x hack/generate-certs.sh
./hack/generate-certs.sh my-project-system
```

<aside class="note" role="note">

<p class="note-title">Namespace</p>

Set the namespace argument to match the `namePrefix` + `-system` in your
`config/default/kustomization.yaml`. For example, if your `namePrefix` is
`my-project-`, run the script with `my-project-system`.

</aside>

---

## Step 2: Create the TLS Secret in the cluster

The Kubebuilder scaffold expects the certificate secret to be named
`webhook-server-cert` in the controller namespace.

```bash
NAMESPACE="my-project-system"

kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret tls webhook-server-cert \
  --cert=config/webhook/certs/tls.crt \
  --key=config/webhook/certs/tls.key \
  --namespace=${NAMESPACE} \
  --dry-run=client -o yaml | kubectl apply -f -
```

---

## Step 3: Disable cert-manager in the Kustomize configuration

Open `config/default/kustomization.yaml` and make the following changes:

**4a. Remove (or comment out) the `../certmanager` resource:**

```yaml
resources:
- ../crd
- ../rbac
- ../manager
- ../webhook
# [CERTMANAGER] Comment out the line below to disable cert-manager
#- ../certmanager
```

**4b. Remove (or comment out) all `replacements` blocks** that reference
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

## Step 4: Patch the webhook `caBundle`

Without cert-manager's CA injector, you must inject the CA bundle manually so
that the Kubernetes API server can verify the webhook's TLS certificate.

**4a. Base64-encode the CA certificate:**

```bash
CA_BUNDLE=$(base64 -w 0 < config/webhook/certs/ca.crt)
echo $CA_BUNDLE   # save this value, you'll need it below
```

**4b. Create a Kustomize JSON patch file** at
`config/webhook/cainjection_patch.yaml`:

```yaml
# config/webhook/cainjection_patch.yaml
# Patch the caBundle for MutatingWebhookConfiguration
- op: add
  path: /webhooks/0/clientConfig/caBundle
  value: <BASE64_CA_BUNDLE>
```

Replace `<BASE64_CA_BUNDLE>` with the output from Step 4a.

If you also have a `ValidatingWebhookConfiguration`, add a second patch file
`config/webhook/cainjection_validating_patch.yaml` with the same content.

**4c. Register the patch in `config/webhook/kustomization.yaml`:**

```yaml
resources:
- manifests.yaml
- service.yaml

patches:
- path: cainjection_patch.yaml
  target:
    group: admissionregistration.k8s.io
    version: v1
    kind: MutatingWebhookConfiguration
# Uncomment if you also have a ValidatingWebhookConfiguration
#- path: cainjection_validating_patch.yaml
#  target:
#    group: admissionregistration.k8s.io
#    version: v1
#    kind: ValidatingWebhookConfiguration
```

---

## Step 5: Build and deploy

```bash
# Build and push your controller image
make docker-build docker-push IMG=<your-registry>/my-project:dev

# Deploy (cert-manager is not needed)
make deploy IMG=<your-registry>/my-project:dev
```

Verify the webhook pod is running and the certificate secret was mounted:

```bash
kubectl get pods -n ${NAMESPACE}
kubectl describe secret webhook-server-cert -n ${NAMESPACE}
```

---

## Alternative: `make run` (local process, not in-cluster)

When running the controller locally with `make run`, the webhook server needs
TLS certificates on the local filesystem. Point controller-runtime to the
certificates generated in Step 1:

```bash
make run ARGS="--cert-dir=config/webhook/certs"
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
