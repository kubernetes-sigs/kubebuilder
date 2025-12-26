# Kubebuilder Project Roadmap 2025

## Ensure Webhook Implementation Stability and Enhance User Experience

**Status:** Partially Complete

### Objective
Enhance the webhooks implementation and user experience.

### Context
The current implementations for webhook conversion and defaulting are stable and tested through basic end-to-end (E2E) workflows.
However, webhook conversion is incomplete, and several bugs need to be addressed. Additionally, the user experience
is hindered by limitations such as the inability to add additional webhooks for same API without using the force flag
and losing their existing customizations on top.

### Goals and Needs
- **CA Injection**: ✅ Complete (Changes available from release `4.4.0`) Ensure that CA injection for conversion webhooks is limited to the relevant Custom Resource (CR) conversions.
  - [GitHub Issue](https://github.com/kubernetes-sigs/kubebuilder/issues/4285)
  - [Pull Request](https://github.com/kubernetes-sigs/kubebuilder/pull/4282)

- **Scaffolding Multiple Webhooks**: Allow adding additional webhooks without requiring forced re-scaffolding.
  - [GitHub Issue](https://github.com/kubernetes-sigs/kubebuilder/issues/4146)

- **Hub and Spoke Model**: ✅ Complete (Changes available from release `4.4.0`) Integrate a hub-and-spoke model for conversion webhooks to streamline implementation.
  - [GitHub Issue](https://github.com/kubernetes-sigs/kubebuilder/issues/2589)
  - [Pull Request](https://github.com/kubernetes-sigs/kubebuilder/pull/4254)

- **Comprehensive E2E Testing**: ✅ Complete ([Example](https://github.com/kubernetes-sigs/kubebuilder/blob/v4.7.1/testdata/project-v4-with-plugins/test/e2e/e2e_test.go#L284-L296)) Expand end-to-end tests for conversion webhooks to validate not only CA injection but also the conversion process itself.
  - [GitHub Issue](https://github.com/kubernetes-sigs/kubebuilder/issues/4297)

- **E2E Test Scaffolding**: (✅ Complete) Improve the E2E test scaffolds under `test/e2e` to validate conversion behavior beyond CA injection for conversion webhooks.

- **Enhanced Multiversion Tutorial**: (✅ Complete) Add E2E tests for conversion webhooks in the multiversion tutorial to support comprehensive user guidance.
  - [GitHub Issue](https://github.com/kubernetes-sigs/kubebuilder/issues/4255)

---
# Roadmap Document

## Enhance the Helm Chart Plugin

**Status:** ✅ Complete (Released in Kubebuilder v4.10.0, introduced `helm/v2-alpha` plugin which supersedes the previous version and addressed the community feedback.)

### Context
A new plugin to help users scaffold a Helm chart to distribute their solutions is implemented as an experimental
feature on the `master` branch and is currently under development. It's initial version will be released in
the next major version of Kubebuilder.

### Objective
The objective of this effort is to ensure that the Helm chart plugin addresses user needs effectively while
providing a seamless and intuitive experience.

### Goals
- Prevent exposure of webhooks data in the Helm chart values.
- Determine whether and how to include sample files and CR configurations in the Helm chart.
- Enable users to specify the path where the Helm chart will be scaffolded.

### References
- [Milestone Helm](https://github.com/kubernetes-sigs/kubebuilder/milestone/39)
- [Code Implementation](https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/plugins/optional/helm)
- [Sample Under Testdata](https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata/project-v4-with-plugins/dist/chart)

---

## Align Tutorials and Samples with Best Practices Proposed by DeployImage Plugin

**Status:** Partially Complete

### Context
The existing tutorials lack consistency with best practices and the layout proposed by the DeployImage plugin.

### Objective
Align tutorials and sample projects with best practices to improve quality and usability.

### Goals
- **Controller Logic Consistency**: Standardize tutorial controller logic to match the DeployImage plugin’s scaffolded controller, including conditions, finalizers, and status updates.

- **Conditional Status in CronJob Spec**: (✅ Complete) Incorporate conditional status handling in the CronJob spec to reflect best practices.

- **Test Logic Consistency**: Ensure tutorial test logic mirrors the tests scaffolded by the DeployImage plugin, adapting as needed for specific cases.

---

## Provide Solutions to Keep Users Updated with the Latest Changes

**Status:** (✅ feature complete from 4.8.0 )
- Proposal: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/update_action.md
- `kubebuilder alpha update` command implemented. More info: https://book.kubebuilder.io/reference/commands/alpha_update
- AutoUpdate Plugin implemented as v1-alpha. More info: https://book.kubebuilder.io/plugins/available/autoupdate-v1-alpha

### Context
Kubebuilder currently offers a "Help to Upgrade" feature via the `kubebuilder alpha generate` command, but applying updates requires significant manual effort.

### Objective
Develop an opt-in mechanism to notify users and automate updates, reducing manual effort and ensuring alignment with the latest Kubebuilder versions.

### Goals
- Facilitate keeping repositories updated with minimal manual intervention.
- Provide automated notifications and updates inspired by Dependabot.
- Maintain compatibility with new Kubebuilder features, best practices, and bug fixes.

---
