# Artifacts

To test your controllers, you will need to use the tarballs containing the required binaries:

```shell
./bin/k8s/
└── 1.25.0-darwin-amd64
    ├── etcd
    ├── kube-apiserver
    └── kubectl
```

These tarballs are released by [controller-tools](https://github.com/kubernetes-sigs/controller-tools),
and you can find the list of available versions at: [envtest-releases.yaml](https://github.com/kubernetes-sigs/controller-tools/blob/main/envtest-releases.yaml).

When you run `make envtest` or `make test`, the necessary tarballs are downloaded and properly
configured for your project.

<aside class="note">
<h1>Setup ENV TEST tool</h1>

To learn more about the tooling used to configure ENVTEST, which is utilized in the `setup-envtest`
target in the Makefile of projects built with Kubebuilder, see the [README](https://github.com/kubernetes-sigs/controller-runtime/blob/main/tools/setup-envtest/README.md)
of its tooling. Additionally, you can find more information by reviewing the Kubebuilder [ENVTEST][env-test-doc] documentation.

</aside>


<aside class="note warning">
<h1>IMPORTANT: Action Required: Ensure that you no longer use https://storage.googleapis.com/kubebuilder-tools </h1>

**Artifacts provided under [https://storage.googleapis.com/kubebuilder-tools](https://storage.googleapis.com/kubebuilder-tools) are deprecated and Kubebuilder maintainers are no longer able to support, build, or ensure the promotion of these artifacts.**

You will find the [ENVTEST][env-test-doc] binaries available in the new location from k8s release `1.28`, see: [https://github.com/kubernetes-sigs/controller-tools/blob/main/envtest-releases.yaml](https://github.com/kubernetes-sigs/controller-tools/blob/main/envtest-releases.yaml).
Also, binaries to test your controllers after k8s `1.29.3` will no longer be found in the old location.

**New binaries are only promoted in the new location**.

**You should ensure that your projects are using the new location.**
Please ensure you use `setup-envtest` from the controller-runtime `release v0.19.0` to be able to download those.
**This update is fully transparent for Kubebuilder users.**

The artefacts for [ENVTEST][env-test-doc] k8s `1.31` are exclusively available at: [Controller Tools Releases][controller-gen].

You can refer to the Makefile of the Kubebuilder scaffold and observe that the envtest setup is consistently aligned across all controller-runtime releases. Starting from `release-0.19`, it is configured to automatically download the artefact from the correct location, **ensuring that kubebuilder users are not impacted.**

```shell
ENVTEST_K8S_VERSION = 1.31.0
ENVTEST_VERSION ?= release-0.19
...
.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))
```

</aside>

[env-test-doc]: ./envtest.md
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[controller-gen]: https://github.com/kubernetes-sigs/controller-tools/releases
