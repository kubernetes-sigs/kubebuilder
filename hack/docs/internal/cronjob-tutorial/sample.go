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
          securityContext:
            runAsNonRoot: true
            runAsUser: 1000
            seccompProfile:
              type: RuntimeDefault
          containers:
          - name: hello
            image: busybox
            args:
            - /bin/sh
            - -c
            - date; echo Hello from the Kubernetes cluster
            securityContext:
              allowPrivilegeEscalation: false
              capabilities:
                drop:
                - ALL
              readOnlyRootFilesystem: false
          restartPolicy: OnFailure`

const certManagerForMetrics = `# - source: # Uncomment the following block to enable certificates for metrics
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
#     - select: # Uncomment the following to set the Service name for TLS config in Prometheus ServiceMonitor
#         kind: ServiceMonitor
#         group: monitoring.coreos.com
#         version: v1
#         name: controller-manager-metrics-monitor
#       fieldPaths:
#         - spec.endpoints.0.tlsConfig.serverName
#       options:
#         delimiter: '.'
#         index: 0
#         create: true

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
#     - select: # Uncomment the following to set the Service namespace for TLS in Prometheus ServiceMonitor
#         kind: ServiceMonitor
#         group: monitoring.coreos.com
#         version: v1
#         name: controller-manager-metrics-monitor
#       fieldPaths:
#         - spec.endpoints.0.tlsConfig.serverName
#       options:
#         delimiter: '.'
#         index: 1
#         create: true`
