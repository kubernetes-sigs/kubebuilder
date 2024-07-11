# Artifacts

# Artifacts

<aside class="note warning">
<h1>IMPORTANT: Kubebuilder no longer produces artifacts</h1>

Kubebuilder has been building those artifacts binaries to allow users
to use the [ENV TEST][env-test-doc] functionality provided by [controller-runtime][controller-runtime]
for several years. However, Google Cloud Platform has [deprecated the Container Registry](https://cloud.google.com/artifact-registry/docs/transition/transition-from-gcr),
which has been used to build and promote these binaries tarballs.

Additionally, ongoing changes and the phase-out of the previous GCP infrastructure mean
that **Kubebuilder maintainers are no longer able to build or ensure the promotion of these binaries.**

Therefore, since those have been building to allow the controller-runtime
[ENV TEST][env-test-doc] library to work, it has been started to be built by [controller-runtime][controller-runtime] itself
under the [controller-gen releases page][controller-gen]. From [controller-runtime][controller-runtime]
release `v0.19.0` the binaries will begin to be pulled out from this page instead.
For more information, see the PR that introduces this change [here](https://github.com/kubernetes-sigs/controller-runtime/pull/2811).

</aside>


Kubebuilder publishes test binaries and container images in addition
to the main binary releases.

## **(Deprecated)** - Test Binaries (Used by ENV TEST)

You can find test binary tarballs for all Kubernetes versions and host platforms at `https://go.kubebuilder.io/test-tools`.
You can find a test binary tarball for a particular Kubernetes version and host platform at `https://go.kubebuilder.io/test-tools/${version}/${os}/${arch}`.

<aside class="note">
<h1>Setup ENV TEST tool</h1>
To know more about the tooling used to configure ENVTEST which is used in the setup-envtest target in the Makefile
of the projects build with Kubebuilder see the [README][readme]
of its tooling.
</aside>


[env-test-doc]: ./envtest.md
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[controller-gen]: https://github.com/kubernetes-sigs/controller-tools/releases
[readme]: https://github.com/kubernetes-sigs/controller-runtime/blob/main/tools/setup-envtest/README.md