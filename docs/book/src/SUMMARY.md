# Summary

[Introduction](./introduction.md)

[Quick Start](./quick-start.md)

[Architecture](./architecture.md)

---

- [Tutorial: Building CronJob](cronjob-tutorial/cronjob-tutorial.md)

  - [What's in a basic project?](./cronjob-tutorial/basic-project.md)
  - [Every journey needs a start, every program a main](./cronjob-tutorial/empty-main.md)
  - [Groups and Versions and Kinds, oh my!](./cronjob-tutorial/gvks.md)
  - [Adding a new API](./cronjob-tutorial/new-api.md)
  - [Designing an API](./cronjob-tutorial/api-design.md)

    - [A Brief Aside: What's the rest of this stuff?](./cronjob-tutorial/other-api-files.md)

  - [What's in a controller?](./cronjob-tutorial/controller-overview.md)
  - [Implementing a controller](./cronjob-tutorial/controller-implementation.md)

    - [You said something about main?](./cronjob-tutorial/main-revisited.md)

  - [Implementing defaulting/validating webhooks](./cronjob-tutorial/webhook-implementation.md)
  - [Running and deploying the controller](./cronjob-tutorial/running.md)

    - [Deploying the cert manager](./cronjob-tutorial/cert-manager.md)
    - [Deploying webhooks](./cronjob-tutorial/running-webhook.md)

  - [Writing tests](./cronjob-tutorial/writing-tests.md)

  - [Epilogue](./cronjob-tutorial/epilogue.md)

- [Tutorial: Multi-Version API](./multiversion-tutorial/tutorial.md)

  - [Changing things up](./multiversion-tutorial/api-changes.md)
  - [Hubs, spokes, and other wheel metaphors](./multiversion-tutorial/conversion-concepts.md)
  - [Implementing conversion](./multiversion-tutorial/conversion.md)

    - [and setting up the webhooks](./multiversion-tutorial/webhooks.md)

  - [Deployment and Testing](./multiversion-tutorial/deployment.md)

- [Tutorial: Component Config](./component-config-tutorial/tutorial.md)

  - [Changing things up](./component-config-tutorial/api-changes.md)
  - [Defining your Config](./component-config-tutorial/define-config.md)

  - [Using a custom type](./component-config-tutorial/custom-type.md)

    - [Adding a new Config Type](./component-config-tutorial/config-type.md)
    - [Updating main](./component-config-tutorial/updating-main.md)
    - [Defining your Custom Config](./component-config-tutorial/define-custom-config.md)

---

- [Migrations](./migrations.md)

  - [Kubebuilder v1 vs v2](./migration/v1vsv2.md)

    - [Migration Guide](./migration/legacy/migration_guide_v1tov2.md)

  - [Kubebuilder v2 vs v3](./migration/v2vsv3.md)

    - [Migration Guide](./migration/migration_guide_v2tov3.md)
    - [Migration by updating the files](./migration/manually_migration_guide_v2_v3.md)

  - [Single Group to Multi-Group](./migration/multi-group.md)

---

- [Reference](./reference/reference.md)

  - [Generating CRDs](./reference/generating-crd.md)
  - [Using Finalizers](./reference/using-finalizers.md)
  - [Watching Resources](./reference/watching-resources.md)
    - [Resources Managed by the Operator](./reference/watching-resources/operator-managed.md)
    - [Externally Managed Resources](./reference/watching-resources/externally-managed.md)
  - [Kind cluster](reference/kind.md)
  - [What's a webhook?](reference/webhook-overview.md)
    - [Admission webhook](reference/admission-webhook.md)
    - [Webhooks for Core Types](reference/webhook-for-core-types.md)
  - [Markers for Config/Code Generation](./reference/markers.md)

    - [CRD Generation](./reference/markers/crd.md)
    - [CRD Validation](./reference/markers/crd-validation.md)
    - [CRD Processing](./reference/markers/crd-processing.md)
    - [Webhook](./reference/markers/webhook.md)
    - [Object/DeepCopy](./reference/markers/object.md)
    - [RBAC](./reference/markers/rbac.md)

  - [controller-gen CLI](./reference/controller-gen.md)
  - [completion](./reference/completion.md)
  - [Artifacts](./reference/artifacts.md)
  - [Platform Support](./reference/platform.md)
  - [Configuring EnvTest](./reference/envtest.md)

  - [Metrics](./reference/metrics.md)

    - [Reference](./reference/metrics-reference.md)

  - [Makefile Helpers](./reference/makefile-helpers.md)
  - [Project config](./reference/project-config.md)

---

- [Plugins][plugins]

  - [Available Plugins](./plugins/available-plugins.md)
    - [To create a project](./docs/invalid)
      - [go/v2 (Deprecated)](./plugins/go-v2-plugin.md)
      - [go/v3 (Default init scaffold)](./plugins/go-v3-plugin.md)
      - [go/v4-alpha](./plugins/go-v4-plugin.md)
    - [To add optional features](./docs/invalid)
      - [declarative/v1](./plugins/declarative-v1.md)
      - [grafana/v1-alpha](./plugins/grafana-v1-alpha.md)
      - [deploy-image/v1-alpha](./plugins/deploy-image-plugin-v1-alpha.md)
    - [To be extended for others tools](./docs/invalid)
      - [kustomize/v1](./plugins/kustomize-v1.md)
      - [kustomize/v2-alpha](./plugins/kustomize-v2-alpha.md)
  - [Extending the CLI](./plugins/extending-cli.md)
  - [Creating your own plugins](./plugins/creating-plugins.md)
  - [Testing your own plugins](./plugins/testing-plugins.md)
  - [Plugins Versioning](./plugins/plugins-versioning.md)

---

[Appendix: The TODO Landing Page](./TODO.md)

[plugins]: ./plugins/plugins.md
