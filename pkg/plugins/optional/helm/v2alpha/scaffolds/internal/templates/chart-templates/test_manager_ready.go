/*
Copyright 2026 The Kubernetes Authors.

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

package charttemplates

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &TestManagerReady{}

// TestManagerReady scaffolds a Helm test that verifies the manager deployment is healthy.
//
// The test validates deployment readiness in two steps:
//  1. Manager deployment is Available (waits up to 5 minutes)
//  2. Manager pod is Running (validates pod actually started)
//
// The test pod uses a dedicated ServiceAccount, Role, and RoleBinding
// and follows security best practices with a restrictive security context.
type TestManagerReady struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// OutputDir specifies the output directory for the chart
	OutputDir string

	// HasWebhooks is currently unused but reserved for future webhook-specific test enhancements
	HasWebhooks bool

	// Force if true allows overwriting the scaffolded file
	Force bool
}

// SetTemplateDefaults implements machinery.Template
func (f *TestManagerReady) SetTemplateDefaults() error {
	if f.Path == "" {
		outputDir := f.OutputDir
		if outputDir == "" {
			outputDir = defaultOutputDir
		}
		f.Path = filepath.Join(outputDir, "chart", "templates", "tests", "test-manager-ready.yaml")
	}

	prefix := f.ProjectName
	f.TemplateBody = fmt.Sprintf(testManagerReadyTemplate,
		prefix, prefix, // ServiceAccount metadata (name, resourceName)
		prefix, prefix, // Role metadata (name, resourceName)
		prefix, prefix, prefix, prefix, // RoleBinding (name, resourceName, roleRef, subject)
		prefix, prefix, prefix, // Pod (name, resourceName, serviceAccountName)
	)

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.SkipFile
	}

	return nil
}

const testManagerReadyTemplate = `---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
    helm.sh/chart: {{ "{{ .Chart.Name }}-{{ .Chart.Version | replace \"+\" \"_\" }}" }}
    app.kubernetes.io/instance: {{ "{{ .Release.Name }}" }}
  annotations:
    "helm.sh/hook": test
    "helm.sh/hook-weight": "-5"
    "helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded
  name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-manager-ready\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
    helm.sh/chart: {{ "{{ .Chart.Name }}-{{ .Chart.Version | replace \"+\" \"_\" }}" }}
    app.kubernetes.io/instance: {{ "{{ .Release.Name }}" }}
  annotations:
    "helm.sh/hook": test
    "helm.sh/hook-weight": "-4"
    "helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded
  name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-manager-ready\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
rules:
  - apiGroups:
      - apps
    resources:
      - deployments
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
      - list
      - watch
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
    helm.sh/chart: {{ "{{ .Chart.Name }}-{{ .Chart.Version | replace \"+\" \"_\" }}" }}
    app.kubernetes.io/instance: {{ "{{ .Release.Name }}" }}
  annotations:
    "helm.sh/hook": test
    "helm.sh/hook-weight": "-3"
    "helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded
  name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-manager-ready\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-manager-ready\" \"context\" $) }}" }}
subjects:
  - kind: ServiceAccount
    name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-manager-ready\" \"context\" $) }}" }}
    namespace: {{ "{{ .Release.Namespace }}" }}
---
apiVersion: v1
kind: Pod
metadata:
  labels:
    app.kubernetes.io/managed-by: {{ "{{ .Release.Service }}" }}
    app.kubernetes.io/name: {{ "{{ include \"%s.name\" . }}" }}
    helm.sh/chart: {{ "{{ .Chart.Name }}-{{ .Chart.Version | replace \"+\" \"_\" }}" }}
    app.kubernetes.io/instance: {{ "{{ .Release.Name }}" }}
  annotations:
    "helm.sh/hook": test
    "helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded
  name: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-manager-ready\" \"context\" $) }}" }}
  namespace: {{ "{{ .Release.Namespace }}" }}
spec:
  restartPolicy: Never
  serviceAccountName: {{ "{{ include \"%s.resourceName\" (dict \"suffix\" \"test-manager-ready\" \"context\" $) }}" }}
  volumes:
    - name: tmp
      emptyDir: {}
  containers:
    - name: test
      image: {{ "{{ .Values.test.image.repository }}:{{ .Values.test.image.tag }}" }}
      imagePullPolicy: {{ "{{ .Values.test.image.pullPolicy }}" }}
      volumeMounts:
        - name: tmp
          mountPath: /tmp
      command:
        - /bin/sh
        - -ec
        - |
          echo "=================================="
          echo "Helm Chart Deployment Test"
          echo "=================================="
          echo "Release: {{ "{{ .Release.Name }}" }}"
          echo "Namespace: {{ "{{ .Release.Namespace }}" }}"
          echo "Chart: {{ "{{ .Chart.Name }}-{{ .Chart.Version }}" }}"
          echo ""
          
          # Check if kubectl is already available in the image
          if command -v kubectl >/dev/null 2>&1; then
            echo "Using kubectl from image (pre-installed)"
          else
            # Download kubectl from official Kubernetes release
            KUBECTL_VERSION="{{ "{{ .Values.test.kubectlVersion }}" }}"
            case "$(uname -m)" in
              x86_64) ARCH="amd64" ;;
              aarch64) ARCH="arm64" ;;
              armv7l) ARCH="arm" ;;
              *) ARCH="$(uname -m)" ;;
            esac
            echo "Downloading kubectl ${KUBECTL_VERSION} for ${ARCH}..."
            wget -q -O /tmp/kubectl "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/${ARCH}/kubectl"
            chmod +x /tmp/kubectl
            export PATH="/tmp:$PATH"
            echo "kubectl ${KUBECTL_VERSION} installed successfully"
          fi
          echo ""

          echo "Step 1/2: Verifying manager deployment is available..."
          if ! kubectl wait --for=condition=Available deployment \
            -l control-plane=controller-manager \
            -n {{ "{{ .Release.Namespace }}" }} --timeout=5m; then
            echo "ERROR: Manager deployment failed to become available"
            echo ""
            echo "Deployment status:"
            kubectl get deployments -n {{ "{{ .Release.Namespace }}" }} -l control-plane=controller-manager || true
            echo ""
            echo "Pod status:"
            kubectl get pods -n {{ "{{ .Release.Namespace }}" }} -l control-plane=controller-manager || true
            echo ""
            echo "Pod details:"
            kubectl describe pods -n {{ "{{ .Release.Namespace }}" }} -l control-plane=controller-manager || true
            exit 1
          fi
          echo "SUCCESS: Manager deployment is available"
          echo ""

          echo "Step 2/2: Verifying manager pod is running..."
          POD_STATUS=$(kubectl get pods -l control-plane=controller-manager \
            -n {{ "{{ .Release.Namespace }}" }} \
            -o jsonpath='{.items[0].status.phase}' 2>/dev/null || echo "")
          
          if [ -z "$POD_STATUS" ]; then
            echo "ERROR: No manager pod found"
            kubectl get pods -n {{ "{{ .Release.Namespace }}" }} -l control-plane=controller-manager || true
            exit 1
          fi

          if [ "$POD_STATUS" != "Running" ]; then
            echo "ERROR: Manager pod is not running (status: $POD_STATUS)"
            echo ""
            echo "Pod status:"
            kubectl get pods -n {{ "{{ .Release.Namespace }}" }} -l control-plane=controller-manager || true
            echo ""
            echo "Pod details:"
            kubectl describe pods -n {{ "{{ .Release.Namespace }}" }} -l control-plane=controller-manager || true
            echo ""
            echo "Pod logs:"
            kubectl logs -n {{ "{{ .Release.Namespace }}" }} -l control-plane=controller-manager --tail=50 || true
            exit 1
          fi
          echo "SUCCESS: Manager pod is running"
          echo ""

          echo "=================================="
          echo "All tests passed successfully!"
          echo "=================================="
      securityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop:
            - ALL
        readOnlyRootFilesystem: true
        runAsNonRoot: true
        runAsUser: 65532
        seccompProfile:
          type: RuntimeDefault
`
