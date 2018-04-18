# Contributing guidelines

## Sign the CLA

Kubernetes projects require that you sign a Contributor License Agreement (CLA) before we can accept your pull requests.
Please see https://git.k8s.io/community/CLA.md for more info

### Contributing A Patch

1. Submit an issue describing your proposed change to the repo in question.
1. The [repo owners](OWNERS) will respond to your issue promptly.
1. If your proposed change is accepted, and you haven't already done so, sign a Contributor License Agreement (see details above).
1. Fork the desired repo, develop and test your code changes.
1. Submit a pull request.

## How to build kubebuilder locally

Setup:

- Download [google container builder](https://cloud.google.com/container-builder/docs/build-debug-locally)
- Export `GOOS` (darwin/linux) and `GOARCH` (amd64) vars to match the system to build
- Export `OUTPUT` to a location to write the directory containing the final build artifacts

```sh
export GOOS=darwin
export GOARCH=amd64
export OUTPUT=/tmp/kubebuilder
```

Run container-builder:

```sh
container-builder-local --config=build/cloudbuild_local.yaml --dryrun=false \
  --substitutions=_GOOS=$GOOS,_GOARCH=$GOARCH --write-workspace=$OUTPUT .
```

Extract `tar.gz` from $OUTPUT to /usr/local

## Running kubebuilder tests

```sh
go test ./pkg/...
```

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-kubebuilder)

### Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).

[google container builder]: https://github.com/kubernetes-sigs/container-builder-local
