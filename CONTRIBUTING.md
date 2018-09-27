# Contributing guidelines

## Sign the CLA

Kubernetes projects require that you sign a Contributor License Agreement (CLA) before we can accept your pull requests.

Please see https://git.k8s.io/community/CLA.md for more info.

## Contributing steps

1. Submit an issue describing your proposed change to the repo in question.
1. The [repo owners](OWNERS) will respond to your issue promptly.
1. If your proposed change is accepted, and you haven't already done so, sign a Contributor License Agreement (see details above).
1. Fork the desired repo, develop and test your code changes.
1. Submit a pull request.

## How to build kubebuilder locally

1. Setup tools
    1. Download and setup [gcloud](https://cloud.google.com/sdk/docs/downloads-interactive) 
    1. Install `cloud-build-local` via `gcloud` 
        ```bash
        $ gcloud components install cloud-build-local
        ```
1. Build
    1. Export `OUTPUT` to a location to write the directory containing the final build artifacts
        ```sh
        $ export OUTPUT=/tmp/kubebuilder
        ```
    2. Run container-builder:
        ```sh
        $ cloud-build-local --config=build/cloudbuild_local.yaml --dryrun=false \
          --write-workspace=$OUTPUT .
        ```
    1. Extract `tar.gz` from $OUTPUT to /usr/local
1. Test
    ```sh
    go test ./pkg/...
    ```

## Community, discussion and support

Learn how to engage with the Kubernetes community on the [community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [Slack](http://slack.k8s.io/)
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-kubebuilder)

## Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
