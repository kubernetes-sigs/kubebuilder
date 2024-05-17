# Summary

[Introduction](./introduction.md)

[Architecture](./architecture.md)

[Quick Start](./quick-start.md)

[Getting Started](./getting-started.md)

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

    - [Deploying cert-manager](./cronjob-tutorial/cert-manager.md)
    - [Deploying webhooks](./cronjob-tutorial/running-webhook.md)

  - [Writing tests](./cronjob-tutorial/writing-tests.md)

  - [Epilogue](./cronjob-tutorial/epilogue.md)

- [Tutorial: Multi-Version API](./multiversion-tutorial/tutorial.md)

  - [Changing things up](./multiversion-tutorial/api-changes.md)
  - [Hubs, spokes, and other wheel metaphors](./multiversion-tutorial/conversion-concepts.md)
  - [Implementing conversion](./multiversion-tutorial/conversion.md)

    - [and setting up the webhooks](./multiversion-tutorial/webhooks.md)

  - [Deployment and Testing](./multiversion-tutorial/deployment.md)

---

- [Migrations](./migrations.md)

  - [Legacy (before <= v3.0.0)](./migration/legacy.md)
    - [Kubebuilder v1 vs v2](migration/legacy/v1vsv2.md)

      - [Migration Guide](./migration/legacy/migration_guide_v1tov2.md)

    - [Kubebuilder v2 vs v3](migration/legacy/v2vsv3.md)

      - [Migration Guide](migration/legacy/migration_guide_v2tov3.md)
      - [Migration by updating the files](migration/legacy/manually_migration_guide_v2_v3.md)
  - [From v3.0.0 with plugins](./migration/v3-plugins.md)
    - [go/v3 vs go/v4](migration/v3vsv4.md)

      - [Migration Guide](migration/migration_guide_gov3_to_gov4.md)
      - [Migration by updating the files](migration/manually_migration_guide_gov3_to_gov4.md)
  - [Single Group to Multi-Group](./migration/multi-group.md)

- [Project Upgrade Assistant](./reference/rescaffold.md)

---

- [Reference](./reference/reference.md)

  - [Generating CRDs](./reference/generating-crd.md)
  - [Using Finalizers](./reference/using-finalizers.md)
  - [Good Practices](./reference/good-practices.md)
  - [Raising Events](./reference/raising-events.md)
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

  - [Sub-Module Layouts](./reference/submodule-layouts.md)
  - [Using an external Type / API](./reference/using_an_external_type.md)

  - [Configuring EnvTest](./reference/envtest.md)

  - [Metrics](./reference/metrics.md)

    - [Reference](./reference/metrics-reference.md)

  - [Makefile Helpers](./reference/makefile-helpers.md)
  - [Project config](./reference/project-config.md)

---

- [Plugins][plugins]

  - [Available Plugins](./plugins/available-plugins.md)
    - [To scaffold a project](./plugins/to-scaffold-project.md)
      - [go/v4 (Default init scaffold)](./plugins/go-v4-plugin.md)
    - [To add optional features](./plugins/to-add-optional-features.md)
      - [grafana/v1-alpha](./plugins/grafana-v1-alpha.md)
      - [deploy-image/v1-alpha](./plugins/deploy-image-plugin-v1-alpha.md)
    - [To be extended for others tools](./plugins/to-be-extended.md)
      - [kustomize/v2 (Default init scaffold with go/v4)](./plugins/kustomize-v2.md)
  - [Extending the CLI](./plugins/extending-cli.md)
  - [Creating your own plugins](./plugins/creating-plugins.md)
  - [Testing your own plugins](./plugins/testing-plugins.md)
  - [Plugins Versioning](./plugins/plugins-versioning.md)
  - [Creating external plugins](./plugins/external-plugins.md)

---

[FAQ](./faq.md)

[Appendix: The TODO Landing Page](./TODO.md)

[plugins]: ./plugins/plugins.md
