# Release Process

The Kubebuilder Project is released on an as-needed basis. The process is as follows:

1. An issue is proposing a new release with a changelog since the last release
1. All [OWNERS](OWNERS) must LGTM this release
1. An OWNER runs `git tag -s $VERSION` and inserts the changelog and pushes the tag with `git push $VERSION`
1. The release issue is closed
1. An announcement email is sent to `kubernetes-kubebuilder@googlegroups.com` with the subject `[ANNOUNCE] kubebuilder $VERSION is released`

Note: This process does not apply to EAP or alpha (pre-)releases which may be cut at any time for development
and testing.

## HEAD releases

The binaries releases for HEAD are available here:

- [kubebuilder-release-master-head-darwin-amd64.tar.gz](https://storage.googleapis.com/kubebuilder-release/kubebuilder-release-master-head-darwin-amd64.tar.gz)
- [kubebuilder-release-master-head-linux-amd64.tar.gz](https://storage.googleapis.com/kubebuilder-release/kubebuilder-release-master-head-linux-amd64.tar.gz)