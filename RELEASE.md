# Release Process

The Kubebuilder Project is released on an as-needed basis. The process is as follows:

**Note:** Releases are done from the `release-MAJOR.MINOR` branches. For PATCH releases it is not required
to create a new branch. Instead, you will just need to ensure that all major fixes are cherry-picked into the respective
`release-MAJOR.MINOR` branch. To know more about versioning check https://semver.org/.

**Note:** Before `3.5.*` release this project was released based on `MAJOR`. A change to the
process was done to ensure that we have an aligned process under the org (similar to `controller-runtime` and
`controller-tools`) and to make it easier to produce patch releases.

## How to do a release

### Create the new branch and the release tag

1. Create a new branch `git checkout -b release-<MAJOR.MINOR>` from master
2. Push the new branch to the remote repository

### Now, let's generate the changelog

1. Create the changelog from the new branch `release-<MAJOR.MINOR>` (`git checkout release-<MAJOR.MINOR>`).
   You will need to use the [kubebuilder-release-tools][kubebuilder-release-tools] to generate release notes. See [here][release-notes-generation]

> **Note**
> - You will need to have checkout locally from the remote repository the previous branch
> - Also, ensure that you fetch all tags from the remote `git fetch --all --tags`
> - Also, if you face issues to generate the release notes you might want to able to sort it out by running i.e.:
> `go run sigs.k8s.io/kubebuilder-release-tools/notes --use-upstream=false --from=v3.11.0 --branch=release-X`


### Draft a new release from GitHub

1. Create a new tag with the correct version from the new `release-<MAJOR.MINOR>` branch
2. Verify the Release Github Action. It should build the assets and publish in the draft release
3. You also need to manually add the changelog generated above on the release page and publish it. Now, the code source is released !

### Update the website docs (https://book.kubebuilder.io/quick-start.html)

1. Push a PR to update the `book-v3` branch with the changes of the latest release branch created (`release-<MAJOR.MINOR>`)
2. Ping in the [Kubebuilder Slack channel](https://kubernetes.slack.com/archives/CAR30FCJZ) and ask for reviews.

### When the release be done and the website update: Announce the new release:

1. Announce the new release on the Slack channel, i.e:

````
:announce: Kubebuilder v3.5.0 has been released!
This release includes a Kubernetes dependency bump to v1.24.
For more info, see the release page: https://github.com/kubernetes-sigs/kubebuilder/releases/tag/v3.5.0
 :tada:  Thanks to all our contributors!
````

2. Announce the new release via email is sent to `kubebuilder@googlegroups.com` with the subject `[ANNOUNCE] Kubebuilder $VERSION is released`


## HEAD releases

The binaries releases for HEAD are available here:

- [kubebuilder-release-master-head-darwin-amd64.tar.gz](https://storage.googleapis.com/kubebuilder-release/kubebuilder-release-master-head-darwin-amd64.tar.gz)
- [kubebuilder-release-master-head-linux-amd64.tar.gz](https://storage.googleapis.com/kubebuilder-release/kubebuilder-release-master-head-linux-amd64.tar.gz)

## How the releases are configured

The releases occur in an account in the Google Cloud (See [here](https://console.cloud.google.com/cloud-build/builds?project=kubebuilder)) using Cloud Build.

### To build the Kubebuilder CLI binaries:

A trigger GitHub action [release](.github/workflows/release.yml) is trigged when a new tag is pushed.
This action will caall the job [./build/.goreleaser.yml](./build/.goreleaser.yml).

###  (Deprecated) - To build the Kubebuilder-tools: (Artifacts required to use ENV TEST)

> We no longer build the artifacts and the promotion of those is deprecated. For more info
see: https://github.com/kubernetes-sigs/kubebuilder/discussions/4082

Kubebuilder projects requires artifacts which are used to do test with ENV TEST (when we call `make test` target)
These artifacts can be checked in the service page: https://storage.googleapis.com/kubebuilder-tools

The build is made from the branch [tools-releases](https://github.com/kubernetes-sigs/kubebuilder/tree/tools-releases) and the trigger will call the `build/cloudbuild_tools.yaml` passing
as argument the architecture and the SO that should be used, e.g:

<img width="553" alt="Screenshot 2022-04-30 at 10 15 41" src="https://user-images.githubusercontent.com/7708031/166099666-ae9cd2df-73fe-47f6-a987-464f63df9a19.png">

For further information see the [README](https://github.com/kubernetes-sigs/kubebuilder/blob/tools-releases/README.md).

### (Deprecated) - To build the `kube-rbac-proxy` images:

> We no longer build the images and the promotion of those images is deprecated. For more info
see: https://github.com/kubernetes-sigs/kubebuilder/discussions/3907

These images are built from the project [brancz/kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy).
The projects built with Kubebuilder creates a side container with `kube-rbac-proxy` to protect the Manager.

These images are can be checked in the consolse, see [here](https://console.cloud.google.com/gcr/images/kubebuilder/GLOBAL/kube-rbac-proxy).

The project `kube-rbac-proxy` is in the process to be donated to the k8s org. However, it is going on for a long time and then,
we have no ETA for that to occur. When that occurs we can automate this process. But until there we need to generate these images
by bumping the versions/tags released by `kube-rbac-proxy` on the branch
[kube-rbac-proxy-releases](https://github.com/kubernetes-sigs/kubebuilder/tree/kube-rbac-proxy-releases)
then the `build/cloudbuild_kube-rbac-proxy.yaml` will generate the images.

To check an example, see the pull request [#2578](https://github.com/kubernetes-sigs/kubebuilder/pull/2578).

**Note**: we cannot use the images produced by the project `kube-rbac-proxy` because we need to ensure
to Kubebuilder users that these images will be available.

### (Deprecated) - To build the `gcr.io/kubebuilder/pr-verifier` images:

> We are working on to move all out from GCP Kubebuilder project. For further information see: https://github.com/kubernetes/k8s.io/issues/2647#issuecomment-2111182864

These images are used to verify the PR title and description. They are built from [kubernetes-sigs/kubebuilder-release-tools](https://github.com/kubernetes-sigs/kubebuilder-release-tools/).
In Kubebuilder, we have been using this project via the GitHub action [.github/workflows/verify.yml](.github/workflows/verify.yml)
and not the image, see:

```yaml
  verify:
    name: Verify PR contents
    runs-on: ubuntu-latest
    steps:
    - name: Verifier action
      id: verifier
      uses: kubernetes-sigs/kubebuilder-release-tools@v0.1.1
      with:
        github_token: ${{ secrets.GITHUB_TOKEN }}
```

However, the image should still be built and maintained since other projects under the org might be using them.

[kubebuilder-release-tools]: https://github.com/kubernetes-sigs/kubebuilder-release-tools
[release-notes-generation]: https://github.com/kubernetes-sigs/kubebuilder-release-tools/blob/master/README.md#release-notes-generation
[release-process]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/VERSIONING.md#releasing
