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

# Runs the helper script from the "Webhook with self-signed certificates"
# guide against a stub kubectl, then checks the certificates it generates and
# the patches it would apply.

set -o errexit
set -o nounset
set -o pipefail

script="$(cd "$(dirname "$0")/../../docs/book/src/reference/webhook-self-signed-cert" && pwd)/webhook-certs.sh"
project="my-project"
service="${project}-webhook-service.${project}-system.svc"

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT
mkdir -p "${work}/bin" "${work}/project"

cat > "${work}/bin/kubectl" <<'EOF'
#!/usr/bin/env bash
echo "$*" >> "${KUBECTL_LOG}"
case "$*" in
  "get mutatingwebhookconfiguration -o name")
    echo "mutatingwebhookconfiguration.admissionregistration.k8s.io/my-project-mutating-webhook-configuration" ;;
  "get validatingwebhookconfiguration -o name")
    echo "validatingwebhookconfiguration.admissionregistration.k8s.io/my-project-validating-webhook-configuration" ;;
  "get crd -o name")
    echo "customresourcedefinition.apiextensions.k8s.io/cronjobs.batch.my-project.example.com" ;;
  get\ *webhookconfiguration*)
    printf 'first\nsecond\n' ;;
  get\ customresourcedefinition*)
    echo '{"service":{}}' ;;
  create\ *)
    echo "kind: Placeholder" ;;
esac
EOF
chmod +x "${work}/bin/kubectl"

export KUBECTL_LOG="${work}/kubectl.log"
(cd "${work}/project" && PATH="${work}/bin:${PATH}" "${script}" "${project}")

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

certs="${work}/project/config/webhook/certs"
openssl verify -CAfile "${certs}/ca.crt" "${certs}/tls.crt" >/dev/null ||
  fail "tls.crt is not signed by ca.crt"
openssl x509 -in "${certs}/tls.crt" -noout -text | grep -q "DNS:${service}," ||
  fail "tls.crt has no SAN for ${service}"
openssl x509 -in "${certs}/tls.crt" -noout -text | grep -q "DNS:${service}.cluster.local" ||
  fail "tls.crt has no SAN for ${service}.cluster.local"

grep -q "create secret tls webhook-server-cert .*--namespace=${project}-system" "${KUBECTL_LOG}" ||
  fail "the webhook-server-cert Secret was not created in ${project}-system"

ca_bundle="$(base64 < "${certs}/ca.crt" | tr -d '\n')"
expected=(
  "/webhooks/0/clientConfig/caBundle"
  "/webhooks/1/clientConfig/caBundle"
  "/spec/conversion/webhook/clientConfig/caBundle"
)
for path in "${expected[@]}"; do
  patch="[{\"op\": \"add\", \"path\": \"${path}\", \"value\": \"${ca_bundle}\"}]"
  grep -qF -- "-p=${patch}" "${KUBECTL_LOG}" ||
    fail "no add patch for ${path} with the generated CA bundle"
done
patches="$(grep -c "^patch " "${KUBECTL_LOG}")"
[[ "${patches}" -eq 5 ]] || fail "expected 5 patches, got ${patches}"

echo "webhook-certs.sh: OK"
