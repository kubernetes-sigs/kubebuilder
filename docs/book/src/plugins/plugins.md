# Plugins

Since the `3.0.0` Kubebuilder version, preliminary support for plugins was added. You can [Extend the CLI and Scaffolds][extending-cli] as well. See that when users run the CLI commands to perform the scaffolds, the plugins are used:

- To initialize a project with a chain of global plugins:

```sh
kubebuilder init --plugins=pluginA,pluginB
```

- To perform an optional scaffold using custom plugins:

```sh
kubebuilder create api --plugins=pluginA,pluginB
```

This section details how to extend Kubebuilder and create your plugins following the same layout structures.

<aside class="note">
<h1>Note</h1>

You can check the existing design proposal docs at [Extensible CLI and Scaffolding Plugins: phase 1][plugins-phase1-design-doc] and [Extensible CLI and Scaffolding Plugins: phase 1.5][plugins-phase1-design-doc-1.5] to know more on what is provided by Kubebuilder CLI and API currently.

</aside>

<aside class="note">
<h1>What is about to came next?</h1>

To know more about Kubebuilder's future vision of the Plugins architecture, see the section [Future vision for Kubebuilder Plugins][section-future-vision-plugins].

</aside>

- [Extending the CLI and Scaffolds](extending-cli.md)
- [Creating your own plugins](creating-plugins.md)
- [Testing your plugins](testing-plugins.md)

[plugins-phase1-design-doc]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1.md
[plugins-phase1-design-doc-1.5]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1-5.md
[extending-cli]: extending-cli.md
[section-future-vision-plugins]: https://book.kubebuilder.io/plugins/creating-plugins.html#future-vision-for-kubebuilder-plugins
