#!/usr/bin/env bash

# Copyright 2026 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# ANCHOR: script
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
CA_BUNDLE=$(base64 < "${CERT_DIR}/ca.crt" | tr -d '\n')

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
        kubectl patch "$webhook" --type='json' -p="[{\"op\": \"add\", \"path\": \"/webhooks/$i/clientConfig/caBundle\", \"value\": \"${CA_BUNDLE}\"}]"
    done
done

echo "Injecting CA bundle into ValidatingWebhookConfigurations..."
for webhook in $(kubectl get validatingwebhookconfiguration -o name | grep "${PROJECT_NAME}" || true); do
    WEBHOOKS_COUNT=$(kubectl get "$webhook" -o jsonpath='{range .webhooks[*]}{@.name}{"\n"}{end}' | wc -l)
    for (( i=0; i<WEBHOOKS_COUNT; i++ )); do
        kubectl patch "$webhook" --type='json' -p="[{\"op\": \"add\", \"path\": \"/webhooks/$i/clientConfig/caBundle\", \"value\": \"${CA_BUNDLE}\"}]"
    done
done

echo "Injecting CA bundle into CRD conversion webhooks..."
for crd in $(kubectl get crd -o name | grep "${PROJECT_NAME}" || true); do
    HAS_WEBHOOK=$(kubectl get "$crd" -o jsonpath='{.spec.conversion.webhook.clientConfig}' || true)
    if [ -n "$HAS_WEBHOOK" ]; then
        kubectl patch "$crd" --type='json' -p="[{\"op\": \"add\", \"path\": \"/spec/conversion/webhook/clientConfig/caBundle\", \"value\": \"${CA_BUNDLE}\"}]"
    fi
done

echo "Done!"
# ANCHOR_END: script
