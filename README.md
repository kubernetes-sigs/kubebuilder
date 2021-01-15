[![Build Status](https://travis-ci.org/kubernetes-sigs/kubebuilder.svg?branch=master)](https://travis-ci.org/kubernetes-sigs/kubebuilder "Travis")
[![Go Report Card](https://goreportcard.com/badge/sigs.k8s.io/kubebuilder)](https://goreportcard.com/report/sigs.k8s.io/kubebuilder)

## Kubebuilder (kube-rbac-proxy branch)

For more information on the overall KubeBuilder project, see the [main branch](https://github.com/kubernetes-sigs/kubebuilder).

This is the branch we use to tag the ["kube-rbac-proxy" image][image-ref] from
its upstream quay.io source.

## How this works/how to update

GCP Cloud Build watches this branch.  On every push, it runs the pipeline
defined in [build/cloudbuild_kube-rbac-proxy.yaml][cloudbuild-file], which
grabs the source images from `quay.io/brancz/kube-rbac-proxy` and tags them as
`gcr.io/kubebuilder/kube-rbac-proxy`, with a tag for each arch as well as
a single manifest bundle of:

- amd64
- arm64
- ppc64le
- s390x

There's also a helper script in [build/thirdparty](build/thirdparty) to assist in the process.

To update, simply update the variable at the top of the [cloudbuild file][cloudbuild-file], 
then submit a PR against this branch.

[image-ref]: https://book.kubebuilder.io/reference/artifacts.html#container-images
[cloudbuild-file]: build/cloudbuild_kube-rbac-proxy.yaml
[envtest-ref]: https://book.kubebuilder.io/reference/testing/envtest.html
