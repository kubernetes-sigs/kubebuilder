# Kubebuilder Project Roadmap 2024

### Updating Scaffolding to Align with the Latest changes of controller-runtime

**Status:** ✅ Complete (Changes available from release `4.3.0`)

**Objective:** Update Kubebuilder's controller scaffolding to align with the latest changes
in controller-runtime, focusing on compatibility and addressing recent updates and deprecations
mainly related to webhooks.

**Context:** Kubebuilder's plugin system is designed for stability, yet it depends on controller-runtime,
which is evolving rapidly with versions still under 1.0.0. Notable changes and deprecations,
especially around webhooks, necessitate Kubebuilder's alignment with the latest practices
and functionalities of controller-runtime. We need update the Kubebuilder scaffolding,
samples, and documentation.

**References:**
- [Issue - Deprecations in Controller-Runtime and Impact on Webhooks](https://github.com/kubernetes-sigs/kubebuilder/issues/3721) - An issue detailing the deprecations in controller-runtime that affect Kubebuilder's approach to webhooks.
- [PR - Update to Align with Latest Controller-Runtime Webhook Interface](https://github.com/kubernetes-sigs/kubebuilder/pull/3399) - A pull request aimed at updating Kubebuilder to match controller-runtime's latest webhook interface.
- [PR - Enhancements to Controller Scaffolding for Upcoming Controller-Runtime Changes](https://github.com/kubernetes-sigs/kubebuilder/pull/3723) - A pull request proposing enhancements to Kubebuilder's controller scaffolding in anticipation of upcoming changes in controller-runtime.


#### (New Optional Plugin) Helm Chart Packaging

**Status:** ✅ Complete ( Initial version merged https://github.com/kubernetes-sigs/kubebuilder/pull/4227 - further improvements and contributions are welcome)

**Objective:** We aim to introduce a new plugin for Kubebuilder that packages projects as Helm charts,
facilitating easier distribution and integration of solutions within the Kubernetes ecosystem. For details on this proposal and how to contribute,
see [GitHub Pull Request #3632](https://github.com/kubernetes-sigs/kubebuilder/pull/3632).

**Motivation:** The growth of the Kubernetes ecosystem underscores the need for flexible and
accessible distribution methods. A Helm chart packaging plugin would simplify the distribution of the solutions
and allow easy integrations with common applications used by administrators.

---
### Transition from Google Cloud Platform (GCP) to build and promote binaries and images

**Status:**
- **Kubebuilder CLI**: :white_check_mark: Complete. It has been built using Go releaser. [More info](./../build/.goreleaser.yml)
- **kube-rbac-proxy Images:**  :white_check_mark: Complete. ([More info](https://github.com/kubernetes-sigs/kubebuilder/discussions/3907))
- **EnvTest binaries:** :white_check_mark: Complete Controller-Runtime maintainers are working in a solution to build them out and take the ownership over this one. More info:
  - https://kubernetes.slack.com/archives/C02MRBMN00Z/p1712457941924299
  - https://kubernetes.slack.com/archives/CCK68P2Q2/p1713174342482079
  - Also, see the PR: https://github.com/kubernetes-sigs/controller-runtime/pull/2811
  - It will be available from the next release v0.19.
- **PR Check image:**  🙌 Seeking Contributions to do the required changes - See that the images used to check the PR titles are also build and promoted by the Kubebuilder project in GCP but are from the project: https://github.com/kubernetes-sigs/kubebuilder-release-tools. The plan in this case is to use the e2e shared infrastructure. [More info](https://github.com/kubernetes/k8s.io/issues/2647#issuecomment-2111182864)

**Objective:** Shift Kubernetes (k8s) project infrastructure from GCP to shared infrastructures.
Furthermore, move from the registry `k8s.gcr.io` to `registry.k8s.io`.

**Motivation:** The initiative to move away from GCP aligns with the broader k8s project's
goal of utilizing shared infrastructures. This transition is crucial for ensuring the availability
of the artifacts in the long run and aligning compliance with other projects under the kubernetes-sig org.
[Issue #2647](https://github.com/kubernetes/k8s.io/issues/2647) provides more details on the move.

**Context:** Currently, Google Cloud is used only for:

- **Rebuild and provide the images for kube-rbac-proxy:**

A particular challenge has been the necessity to rebuild images for the
[kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy), which is in the process of being
donated to kubernetes-sig. This transition was expected to eliminate the need for
continuous re-tagging and rebuilding of its images to ensure their availability to users.
The configuration for building these images is outlined
[here](https://github.com/kubernetes-sigs/kubebuilder/blob/master/RELEASE.md#to-build-the-kube-rbac-proxy-images).

- **Build and Promote EnvTest binaries**:

The development of Kubebuilder Tools and EnvTest binaries,
essential for controller tests, represents another area reliant on k8s binaries
traditionally built within GCP environments. Our documentation on building these artifacts is
available [here](https://github.com/kubernetes-sigs/kubebuilder/blob/master/RELEASE.md#to-build-the-kubebuilder-tools-artifacts-required-to-use-env-test).

**We encourage the Kubebuilder community to participate in this discussion, offering feedback and contributing ideas
to refine these proposals. Your involvement is crucial in shaping the future of secure and efficient project scaffolding in Kubebuilder.**

---
### kube-rbac-proxy's Role in Default Scaffold

**Status:** :white_check_mark: Complete

- **Resolution**: The usage of kube-rbac-proxy has been discontinued from the default scaffold. We plan to provide other helpers to protect the metrics endpoint. Furthermore, once the project is accepted under kubernetes-sig or kubernetes-auth, we may contribute to its maintainer in developing an external plugin for use with projects built with Kubebuilder.
   - **Proposal**: [https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/discontinue_usage_of_kube_rbac_proxy.md](https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/discontinue_usage_of_kube_rbac_proxy.md)
   - **PR**: [https://github.com/kubernetes-sigs/kubebuilder/pull/3899](https://github.com/kubernetes-sigs/kubebuilder/pull/3899)
   - **Communication**: [https://github.com/kubernetes-sigs/kubebuilder/discussions/3907](https://github.com/kubernetes-sigs/kubebuilder/discussions/3907)

**Objective:** Evaluate potential modifications or the exclusion of [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy)
from the default Kubebuilder scaffold in response to deprecations and evolving user requirements.

**Context:** [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) , a key component for securing Kubebuilder-generated projects,
faces significant deprecations that impact automatic certificate generation.
For more insights into these challenges, see [Issue #3524](https://github.com/kubernetes-sigs/kubebuilder/issues/3524).

This situation necessitates a reevaluation of its inclusion and potentially prompts users to
adopt alternatives like cert-manager by default. Additionally, the requirement to manually rebuild
[kube-rbac-proxy images—due](https://github.com/kubernetes-sigs/kubebuilder/blob/master/RELEASE.md#to-build-the-kube-rbac-proxy-images)
to its external status from Kubernetes-SIG—places a considerable maintenance
burden on Kubebuilder maintainers.

**Motivations:**
- Address kube-rbac-proxy breaking changes/deprecations.
  - For further information: [Issue #3524 - kube-rbac-proxy warn about deprecation and future breaking changes](https://github.com/kubernetes-sigs/kubebuilder/issues/3524)
- Feedback from the community has highlighted a preference for cert-manager's default integration, aiming security with Prometheus and metrics.
  - More info: [GitHub Issue #3524 - Improve scaffolding of ServiceMonitor](https://github.com/kubernetes-sigs/kubebuilder/issues/3657)
- Desire for kube-rbac-proxy to be optional, citing its prescriptive nature.
  - See: [Issue #3482 - The kube-rbac-proxy is too opinionated to be opt-out.](https://github.com/kubernetes-sigs/kubebuilder/issues/3482)
- Reduce the maintainability effort to generate the images used by Kubebuilder projects and dependency within third-party solutions.
  - Related issues:
    - [Issue #1885 - use a NetworkPolicy instead of kube-rbac-proxy](https://github.com/kubernetes-sigs/kubebuilder/issues/1885)
    - [Issue #3230 - Migrate away from google.com gcp project kubebuilder](https://github.com/kubernetes-sigs/kubebuilder/issues/3230)

---
### Providing Helpers for Project Distribution

#### Distribution via Kustomize

**Status:** :white_check_mark: Complete

- **Resolution**: As of release ([v3.14.0](https://github.com/kubernetes-sigs/kubebuilder/releases/tag/v3.14.0)), Kubebuilder includes enhanced support for project distribution. Users can now scaffold projects with a `build-installer` makefile target. This improvement enables the straightforward deployment of solutions directly to Kubernetes clusters. Users can deploy their projects using commands like:

```shell
kubectl apply -f https://raw.githubusercontent.com/<org>/my-project/<tag or branch>/dist/install.yaml
```
This enhancement streamlines the process of getting Kubebuilder projects running on clusters, providing a seamless deployment experience.

---
### **(Major Release for Kubebuilder CLI 4.x)** Removing Deprecated Plugins for Enhanced Maintainability and User Experience

**Status:** : ✅ Complete - Release was done
  - **Remove Deprecations**:https://github.com/kubernetes-sigs/kubebuilder/issues/3603
  - **Bump Module**: https://github.com/kubernetes-sigs/kubebuilder/pull/3924

**Objective:** To remove all deprecated plugins from Kubebuilder to improve project maintainability and
enhance user experience. This initiative also includes updating the project documentation to provide clear
and concise information, eliminating any confusion for users. **More Info:** [GitHub Discussion #3622](https://github.com/kubernetes-sigs/kubebuilder/discussions/3622)

**Motivation:** By focusing on removing deprecated plugins—specifically, versions or kinds that can no
longer be supported—we aim to streamline the development process and ensure a higher quality user experience.
Clear and updated documentation will further assist in making development workflows more efficient and less prone to errors.

