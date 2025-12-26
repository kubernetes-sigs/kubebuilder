# Kubebuilder Project Roadmap 2026

The main goals for 2026 are to promote adoption of the latest Kubebuilder versions and update automation, align tutorials and samples with best practices proposed by the [DeployImage plugin](https://kubebuilder.io/plugins/available/deploy-image-plugin-v1-alpha), improve documentation quality and consistency, explore how we can better leverage AI capabilities, and strengthen Kubebuilder as an API and plugin framework to encourage the creation and adoption of external plugins that extend and integrate with Kubebuilder.

## Promote and encourage adoption of the latest Kubebuilder versions and automation mechanisms to stay updated

**Status:** WIP

### Context
In 2025, we introduced the `kubebuilder alpha update` command and the AutoUpdate plugin to help users stay current with the latest Kubebuilder changes.
However, adoption of these features has been limited because many users are not aware of them or cannot update their projects easily to
the latest Kubebuilder versions and take advantage of these automation mechanisms.

### Objective
Ensure more projects use the latest Kubebuilder versions and adopt automation mechanisms to stay updated.

### Goals
- **Migration Documentation**:
  - (Status Done but looking for collaboration and follow up) Simplify and enhance migration documentation to guide users through updating their projects to the latest Kubebuilder versions. Highlight the available automation mechanisms and provide a generic guide to migrate from any version to the latest manually, so users can then adopt the automation mechanisms going forward.

- **Create Campaign to Promote Update Features**:
  - (Status TODO) Launch a campaign to raise awareness about the `kubebuilder alpha update` command and the AutoUpdate plugin, highlighting their benefits and encouraging adoption among users.
    - Similar to past campaigns where we created issues for public repos to help users become aware of critical changes (e.g., Kubernetes API deprecation from `v1beta1` to `v1`, and the migration away from `gcr.io/kubebuilder/kube-rbac-proxy`). See [discussion #3907](https://github.com/kubernetes-sigs/kubebuilder/discussions/3907).
    - Note: before running this campaign, ensure migration documentation is in place to support users through the update process.
    - [GitHub Issue](https://github.com/kubernetes-sigs/kubebuilder/issues/5291)

## Align Tutorials and Samples with Best Practices Proposed by DeployImage Plugin

**Status:** TODO

### Context
The existing tutorials lack consistency with best practices and the layout proposed by the DeployImage plugin.

### Objective
Align tutorials and sample projects with best practices to improve quality and usability.

### Goals
- **Controller Logic Consistency**: Standardize tutorial controller logic to match the DeployImage plugin’s scaffolded controller, including conditions, finalizers, and status updates.
  - [GitHub Issue](https://github.com/kubernetes-sigs/kubebuilder/issues/4140)
- **Test Logic Consistency**: Ensure tutorial test logic mirrors the tests scaffolded by the DeployImage plugin, adapting as needed for specific cases.
  - [GitHub Issue](https://github.com/kubernetes-sigs/kubebuilder/issues/5024)

---

## Enhance Webhooks CLI User Experience to support additional Webhook scaffolding without requiring `--force`

**Status:** TODO

### Objective
Enhance the webhooks implementation and user experience.

### Context
Currently, users can scaffold webhooks, but if they want to add additional webhook types for the same API they need to use `--force`.
This may overwrite existing customizations. The goal is to support iterative workflows where users can scaffold webhook type A and later add webhook type B
without requiring forced re-scaffolding. The `--force` flag should remain available.

### Goals and Needs
- **Scaffolding Multiple Webhooks**: Allow adding additional webhook types for the same API without requiring forced re-scaffolding.
  - [GitHub Issue](https://github.com/kubernetes-sigs/kubebuilder/issues/4146)

---

## External plugins examples improvement and promotion

**Status:** TODO

### Objective
Enhance the examples to demonstrate usage of external plugins for end users and encourage the usage of Kubebuilder as an API and plugin framework.
Help projects create their own plugins to extend and integrate with Kubebuilder.

### Context
Kubebuilder supports external plugins, but we need clearer, maintained examples that show how to build, distribute, and use them in real projects.

### Goals and Needs
- **Make sampleexternalplugin a Valid Reference Implementation**
  - [GitHub Issue](https://github.com/kubernetes-sigs/kubebuilder/issues/4146)
