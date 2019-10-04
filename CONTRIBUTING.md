# Contributing guidelines

## Sign the CLA

Kubernetes projects require that you sign a Contributor License Agreement (CLA) before we can accept your pull requests.

Please see https://git.k8s.io/community/CLA.md for more info.

## Prerequisites
- [go](https://golang.org/dl/) version v1.12+.
- [dep](https://github.com/golang/dep) dep v0.5.1+ < v.0.5.4
- [docker](https://docs.docker.com/install/) version 17.03+.
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) version v1.11.3+.
- [kustomize](https://sigs.k8s.io/kustomize/docs/INSTALL.md) v3.1.0+
- Access to a Kubernetes v1.11.3+ cluster.

## Contributing steps
1. Submit an issue describing your proposed change to the repo in question.
1. The [repo owners](OWNERS) will respond to your issue promptly.
1. If your proposed change is accepted, and you haven't already done so, sign a Contributor License Agreement (see details above).
1. Fork the desired repo, develop and test your code changes.
1. Submit a pull request.

## How to build kubebuilder locally

1. Run the following command to clone your fork of the project locally in the dir /src/sigs.k8s.io/kubebuilder

```
$ git clone git@github.com:<user>/kubebuilder.git $GOPATH/src/sigs.k8s.io/kubebuilder
```

1. Ensure you activate module support before continue (`$ export GO111MODULE=on`)
1. Build the project by using the command `make build` 
1. Run the tests by using the command `make test`
1. Run the command `make install` to create a bin with the source code 
1. Run the script `./generated_golden.sh` to update/generate the mock data used in the e2e test in `$GOPATH/src/sigs.k8s.io/kubebuilder/testdata/`

## Community, discussion and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-kubebuilder)

## Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
