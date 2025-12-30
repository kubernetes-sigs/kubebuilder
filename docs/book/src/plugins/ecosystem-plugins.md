# Ecosystem Plugins

This section tracks external plugins developed by the community that integrate
with and extend Kubebuilder. These plugins are maintained by their respective
projects and provide additional functionality beyond Kubebuilder's built-in capabilities.

## Why Ecosystem Plugins?

Kubebuilder's plugin architecture enables third-party projects to extend its
functionality without requiring changes to Kubebuilder itself. This approach:

- Allows specialized tools to integrate seamlessly with Kubebuilder workflows
- Enables maintainers to manage updates aligned with their own release cycles
- Provides users with flexibility to choose the tools that best fit their needs

For information on creating your own external plugin, see
[Creating External Plugins](./extending/external-plugins.md).

## Featured Plugins

The following plugins are known to integrate with Kubebuilder:

### Operator SDK

[Operator SDK](https://github.com/operator-framework/operator-sdk) extends
Kubebuilder to support building operators using Ansible and Helm, in addition
to Go. It also provides integration with the
[Operator Lifecycle Manager (OLM)](https://olm.operatorframework.io/).

**Repository:** https://github.com/operator-framework/operator-sdk

**Features:**
- Ansible-based operator development
- Helm-based operator development
- OLM bundle generation and integration
- Scorecard testing framework

**Usage:**
```sh
# Initialize with Operator SDK
operator-sdk init --plugins=ansible
# or
operator-sdk init --plugins=helm
```

---

## Sample External Plugins

The following sample plugins demonstrate how to build external plugins in
different programming languages:

| Plugin | Language | Description |
|--------|----------|-------------|
| [sampleexternalplugin](https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/simple-external-plugin-tutorial/testdata/sampleexternalplugin/v1) | Go | Reference implementation in the Kubebuilder repository |
| [POC-Phase2-Plugins](https://github.com/rashmigottipati/POC-Phase2-Plugins) | Python | Demonstrates external plugin development in Python |
| [kb-js-plugin](https://github.com/Eileen-Yu/kb-js-plugin) | JavaScript | Demonstrates external plugin development in JavaScript |

---

## Adding Your Plugin

If you have developed an external plugin that integrates with Kubebuilder and
would like it listed here, please submit a pull request to add your plugin
to this page.

**Guidelines for submissions:**
- Your plugin should be publicly available
- Include a brief description of the plugin's purpose
- Provide a link to the repository
- Document basic usage instructions

For questions or support in developing Kubebuilder plugins, reach out to the
community via [Slack](http://slack.k8s.io/) or the
[mailing list](https://groups.google.com/forum/#!forum/kubebuilder).
