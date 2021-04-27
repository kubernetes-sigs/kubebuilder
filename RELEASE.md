# Release Process

The Kubebuilder Project is released on an as-needed basis. The process is as follows:

1. An issue is proposing a new release with a changelog since the last release. You will need to use the [kubebuilder-release-tools][kubebuilder-release-tools] to generate the notes. See [here][release-notes-generation]
1. All [OWNERS](OWNERS) must LGTM this release
1. An OWNER runs `git tag -s $VERSION` and pushes the tag with `git push $VERSION`. Note that after the OWNER push the tag the CI will automatically add the release notes and the assets.
1. A PR needs to be created to merge `master` branch into `book-v3` to pick up the new docs. 
1. The release issue is closed
1. An announcement email is sent to `kubebuilder@googlegroups.com` with the subject `[ANNOUNCE] kubebuilder $VERSION is released`

**Notes:** This process does not apply to EAP or alpha (pre-)releases which may be cut at any time for development
and testing.

For further information about versioning and update the Kubebuilder binaries check the [versioning][release-process] doc.

## HEAD releases

The binaries releases for HEAD are available here:

- [kubebuilder-release-master-head-darwin-amd64.tar.gz](https://storage.googleapis.com/kubebuilder-release/kubebuilder-release-master-head-darwin-amd64.tar.gz)
- [kubebuilder-release-master-head-linux-amd64.tar.gz](https://storage.googleapis.com/kubebuilder-release/kubebuilder-release-master-head-linux-amd64.tar.gz)

[kubebuilder-release-tools]: https://github.com/kubernetes-sigs/kubebuilder-release-tools
[release-notes-generation]: https://github.com/kubernetes-sigs/kubebuilder-release-tools/blob/master/README.md#release-notes-generation
[release-process]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/VERSIONING.md#releasing 