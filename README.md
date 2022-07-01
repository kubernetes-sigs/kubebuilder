[![Build Status](https://travis-ci.org/kubernetes-sigs/kubebuilder.svg?branch=master)](https://travis-ci.org/kubernetes-sigs/kubebuilder "Travis")
[![Go Report Card](https://goreportcard.com/badge/sigs.k8s.io/kubebuilder)](https://goreportcard.com/report/sigs.k8s.io/kubebuilder)

## Kubebuilder (tools branch)

For more information on the overall KubeBuilder project, see the [main branch](https://github.com/kubernetes-sigs/kubebuilder).

This is the branch we use to build the ["KubeBuilder testing tools"
binaries][binaries-ref], which consist of a copy of kubectl, kube-apiserver,
and etcd for use in [integration testing with envtest][envtest-ref].

## How this works/how to update

GCP Cloud Build watches this branch.  On every push, it runs the pipeline defined in [build/cloudbuild_tools.yaml](build/cloudbuild_tools.yaml) once each with the following sets of configuration:

- `_GOARCH=amd64 _GOOS=darwin`
- `_GOARCH=amd64 _GOOS=linux`
- `_GOARCH=arm64 _GOOS=linux`
- `_GOARCH=arm64 _GOOS=darwin`
- `_GOARCH=ppc64le _GOOS=linux`

(we may add more the in the future).

The pipline then collects or builds the relevant binaries, and publishes them to a [GCS bucket](https://go.kubebuilder.io/test-tools).

Each platform has a Dockerfile in [build/thirdparty](build/thirdparty) to assist in the process:

- For Linux, this involves simply downloading the canonical releases from the
  offically Kubernetes and etcd releases & taring them up.

- For Darwin, since the official Kubernetes releases don't build the control
  plane, we instead build kube-apiserver ourselves, but use etcd & kubectl from
  official releases.

To update, simply update all references to the old Kubernetes & etcd versions
across the pipeline YAML & Dockerfiles, then submit a PR against this branch.

[binaries-ref]: https://book.kubebuilder.io/reference/artifacts.html
[envtest-ref]: https://book.kubebuilder.io/reference/testing/envtest.html
