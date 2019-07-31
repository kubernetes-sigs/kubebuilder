# Summary

[Introduction](./introduction.md)

[Quick Start](./quick-start.md)

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

  - [Epilogue](./cronjob-tutorial/epilogue.md)

- [Tutorial: Multi-Version API](./multiversion-tutorial/tutorial.md)

  - [Changing things up](./multiversion-tutorial/api-changes.md)
  - [Hubs, spokes, and other wheel metaphors](./multiversion-tutorial/conversion-concepts.md)
  - [Implementing conversion](./multiversion-tutorial/conversion.md)

      - [and setting up the webhooks](./multiversion-tutorial/webhooks.md)

  - [Deployment and Testing](./multiversion-tutorial/deployment.md)

---

- [Reference](./reference/reference.md)

  - [Generating CRDs](./reference/generating-crd.md)
  - [Using Finalizers](./reference/using-finalizers.md)
  - [Kind cluster](reference/kind.md)
  - [What's a webhook?](reference/webhook-overview.md)
    - [Admission webhook](reference/admission-webhook.md)
  - [Markers for Config/Code Generation](./reference/markers.md)

      - [CRD Generation](./reference/markers/crd.md)
      - [CRD Validation](./reference/markers/crd-validation.md)
      - [Webhook](./reference/markers/webhook.md)
      - [Object/DeepCopy](./reference/markers/object.md)
      - [RBAC](./reference/markers/rbac.md)

---

[Appendix: The TODO Landing Page](./TODO.md)
