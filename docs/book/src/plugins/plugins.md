# Plugins

Kubebuilder's architecture is fundamentally plugin-based.
This design enables the Kubebuilder CLI to evolve while maintaining
backward compatibility with older versions, allowing users to opt-in or
opt-out of specific features, and enabling seamless integration
with external tools.

By leveraging plugins, projects can extend Kubebuilder and use it as a
library to support new functionalities or implement custom scaffolding
tailored to their users' needs. This flexibility allows maintainers
to build on top of Kubebuilder’s foundation, adapting it to specific
use cases while benefiting from its powerful scaffolding engine.

Plugins offer several key advantages:

- **Backward compatibility**: Ensures older layouts and project structures remain functional with newer versions.
- **Customization**: Allows users to opt-in or opt-out for specific features (i.e. [Grafana][grafana-plugin] and [Deploy Image][deploy-image] plugins)
- **Extensibility**: Facilitates integration with third-party tools and projects that wish to provide their own [External Plugins][external-plugins], which can be used alongside Kubebuilder to modify and enhance project scaffolding or introduce new features.

**For example, to initialize a project with multiple global plugins:**

```sh
kubebuilder init --plugins=pluginA,pluginB,pluginC
```

**For example, to apply custom scaffolding using specific plugins:**

```sh
kubebuilder create api --plugins=pluginA,pluginB,pluginC
OR
kubebuilder create webhook --plugins=pluginA,pluginB,pluginC
OR
kubebuilder edit --plugins=pluginA,pluginB,pluginC
```

This section details the available plugins, how to extend Kubebuilder,
and how to create your own plugins while following the same layout structures.

- [Available Plugins](./available-plugins.md)
- [Extending](./extending.md)
- [Plugins Versioning](./plugins-versioning.md)

[extending-cli]: extending.md
[grafana-plugin]: ./available/grafana-v1-alpha.md
[deploy-image]: ./available/deploy-image-plugin-v1-alpha.md
[external-plugins]: ./extending/external-plugins.md