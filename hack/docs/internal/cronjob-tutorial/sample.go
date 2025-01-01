/*
Copyright 2023 The Kubernetes Authors.

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

package cronjob

const cronjobSample = `
  schedule: "*/1 * * * *"
  startingDeadlineSeconds: 60
  concurrencyPolicy: Allow # explicitly specify, but Allow is also default.
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: hello
            image: busybox
            args:
            - /bin/sh
            - -c
            - date; echo Hello from the Kubernetes cluster
          restartPolicy: OnFailure`

const certManagerForMetricsAndWebhooks = `#replacements:
# - source: # Uncomment the following block to enable certificates for metrics
#     kind: Service
#     version: v1
#     name: controller-manager-metrics-service
#     fieldPath: metadata.name
#   targets:
#     - select:
#         kind: Certificate
#         group: cert-manager.io
#         version: v1
#         name: metrics-certs
#       fieldPaths:
#         - spec.dnsNames.0
#         - spec.dnsNames.1
#       options:
#         delimiter: '.'
#         index: 0
#         create: true
#
# - source:
#     kind: Service
#     version: v1
#     name: controller-manager-metrics-service
#     fieldPath: metadata.namespace
#   targets:
#     - select:
#         kind: Certificate
#         group: cert-manager.io
#         version: v1
#         name: metrics-certs
#       fieldPaths:
#         - spec.dnsNames.0
#         - spec.dnsNames.1
#       options:
#         delimiter: '.'
#         index: 1
#         create: true
#
# - source: # Uncomment the following block if you have any webhook
#     kind: Service
#     version: v1
#     name: webhook-service
#     fieldPath: .metadata.name # Name of the service
#   targets:
#     - select:
#         kind: Certificate
#         group: cert-manager.io
#         version: v1
#         name: serving-cert
#       fieldPaths:
#         - .spec.dnsNames.0
#         - .spec.dnsNames.1
#       options:
#         delimiter: '.'
#         index: 0
#         create: true
# - source:
#     kind: Service
#     version: v1
#     name: webhook-service
#     fieldPath: .metadata.namespace # Namespace of the service
#   targets:
#     - select:
#         kind: Certificate
#         group: cert-manager.io
#         version: v1
#         name: serving-cert
#       fieldPaths:
#         - .spec.dnsNames.0
#         - .spec.dnsNames.1
#       options:
#         delimiter: '.'
#         index: 1
#         create: true
#
# - source: # Uncomment the following block if you have a ValidatingWebhook (--programmatic-validation)
#     kind: Certificate
#     group: cert-manager.io
#     version: v1
#     name: serving-cert # This name should match the one in certificate.yaml
#     fieldPath: .metadata.namespace # Namespace of the certificate CR
#   targets:
#     - select:
#         kind: ValidatingWebhookConfiguration
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 0
#         create: true
# - source:
#     kind: Certificate
#     group: cert-manager.io
#     version: v1
#     name: serving-cert
#     fieldPath: .metadata.name
#   targets:
#     - select:
#         kind: ValidatingWebhookConfiguration
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 1
#         create: true
#
# - source: # Uncomment the following block if you have a DefaultingWebhook (--defaulting )
#     kind: Certificate
#     group: cert-manager.io
#     version: v1
#     name: serving-cert
#     fieldPath: .metadata.namespace # Namespace of the certificate CR
#   targets:
#     - select:
#         kind: MutatingWebhookConfiguration
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 0
#         create: true
# - source:
#     kind: Certificate
#     group: cert-manager.io
#     version: v1
#     name: serving-cert
#     fieldPath: .metadata.name
#   targets:
#     - select:
#         kind: MutatingWebhookConfiguration
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 1
#         create: true`
