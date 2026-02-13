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

package scaffolds

// KubernetesVersion is the Kubernetes version used for kubectl in generated Helm chart tests.
// This should match the k8s.io/api version used in testdata projects' go.mod files.
//
// Update this when updating the Kubernetes version in Kubebuilder (see VERSIONING.md).
// The version should correspond to the k8s.io/api version (e.g., k8s.io/api v0.35.0 → v1.35.0)
//
// This constant is used to download kubectl in Helm test pods from the official Kubernetes release:
//   - https://dl.k8s.io/release/${KubernetesVersion}/bin/linux/${ARCH}/kubectl
//
// The Helm test uses busybox:1.37 as the base image and downloads kubectl at runtime to ensure:
//   - Official kubectl binaries from dl.k8s.io
//   - Multi-architecture support (amd64, arm64, arm, ppc64le, s390x)
//   - Minimal and secure base image
//   - Modern TLS support for downloading from dl.k8s.io
//
// GitHub workflows in scaffolded projects extract the version dynamically from go.mod
// and do not use this constant.
const KubernetesVersion = "v1.35.0"
