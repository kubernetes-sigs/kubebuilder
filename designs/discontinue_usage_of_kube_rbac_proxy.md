| Authors         | Creation Date | Status        | Extra |
|-----------------|---------------|---------------|-------|
| @camilamacedo86 | 07/04/2024    | Implementable | -     |

# Discontinue Kube RBAC Proxy in Default Kubebuilder Scaffolding

This proposal highlights the need to reassess the usage of [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy)
in the default scaffold due to the evolving k8s infra and community feedback. Key considerations include the transition to a shared infrastructure requiring
all images to be published on [registry.k8s.io][registry.k8s.io], the deprecation
of Google Cloud Platform's [Container Registry](https://cloud.google.com/artifact-registry/docs/transition/prepare-gcr-shutdown), and the fact
that [kube-rbac-proxy][kube-rbac-proxy] is yet to be part of the Kubernetes ecosystem umbrella.

The dependency on a potentially discontinuable Google infrastructure,
**which is out of our control**, paired with the challenges of maintaining,
building, or promoting [kube-rbac-proxy][kube-rbac-proxy] images,
calls for a change.

In this document is proposed to replace the [kube-rbac-proxy][kube-rbac-proxy] within
[Network Policies][k8s-doc-networkpolicies] follow-up for potentially enhancements
to protect the metrics endpoint combined with [cert-manager][cert-manager] and a new
a feature introduced in controller-runtime, see [here][cr-pr].

**For the future (when kube-rbac-proxy be part of the k8s umbrella)**, it is proposed the usage of the
[Plugins API provided by Kubebuilder](./../docs/book/src/plugins/plugins.md),
to create an [external plugin](./../docs/book/src/plugins/creating-plugins.md)
to properly integrate the solution with Kubebuilder and provide a helper to allow users to opt-in as they please them.

## Open Questions

- 1) [Network Policies][k8s-doc-networkpolicies] is implemented by the cluster’s CNI. Are we confident that all the major CNIs in use support the proposed policy?

> Besides [Network Policies][k8s-doc-networkpolicies] being part of the core Kubernetes API, their enforcement relies on the CNI plugin installed in
the Kubernetes cluster. While support and implementation details vary among CNIs, the most commonly used ones,
such as Calico, Cilium, WeaveNet, and Canal, offer  support for NetworkPolicies.
>
>Also, there was concern in the past because AWS did not support it. However, this changed,
>as detailed in their announcement: [Amazon VPC CNI now supports Kubernetes Network Policies](https://aws.amazon.com/blogs/containers/amazon-vpc-cni-now-supports-kubernetes-network-policies/).
>
>Moreover, under this proposal, users can still disable/enable this option as they please them.

- 2) NetworkPolicy is a simple firewall and does not provide `authn/authz` and encryption.

> Yes, that's correct. NetworkPolicy acts as a basic firewall for pods within a Kubernetes cluster, controlling traffic
> flow at the IP address or port level. However, it doesn't handle authentication (authn), authorization (authz),
> or encryption directly like kube-rbac-proxy solution.
>
> However, if we can combine the cert-manager and the new feature provided
> by controller-runtime, we can achieve the same or a superior level of protection
> without relying on any extra third-party dependency.

- 3) Could not Kubebuilder maintainers use the shared infrastructure to continue building and promoting those images under the new `registry.k8s.io`?

> We tried to do that, see [here](https://github.com/kubernetes/test-infra/blob/master/config/jobs/image-pushing/k8s-staging-kubebuilder.yaml) the recipe implemented.
> However, it does not work because kube-rbac-proxy is not under the
> kubernetes umbrella. Moreover, we experimented with the GitHub Repository as an alternative approach, see the [PR](https://github.com/kubernetes-sigs/kubebuilder/pull/3854) but seems
> that we are not allowed to use it. Nevertheless, neither approach sorts out all motivations and requirements
> Ideally, Kubebuilder should not be responsible for maintaining and promoting third-party artefacts.

- 4) However, is not Kubebuilder also building and promoting the binaries required to be used within [EnvTest](./../docs/book/src/reference/envtest.md)
feature implemented in controller-runtime?

> Yes, but it also will need to change. Controller-runtime maintainers are looking for solutions to
> build those binaries inside its project since it seems part of its domain. This change is likely
> to be transparent to the community users.

- 5) Could we not use the Controller-Runtime feature [controller-runtime][cr-pr] which enable secure metrics serving over HTTPS?

Yes, after some changes are addressed. After we ask for a hand for reviews from skilled auth maintainers and receive feedback, it appears that this configuration needs to
align with best practices. See the [issue](https://github.com/kubernetes-sigs/controller-runtime/issues/2781)
raised to track this need.

- 6) Could we not make [cert-manager][cert-manager] mandatory?

> No, we can not. One of the goals of Kubebuilder is to make it easier for new
users. So, we cannot make mandatory the usage of a third party as cert-manager
for users by default and to only quick-start.
>
> However, we can make mandatory the usage of
[cert-manager][cert-manager] for some specific features like use kube-rbac-proxy
or, as it is today, using webhooks, a more advanced and optional option.

## Summary

Starting with release `3.15.0`, Kubebuilder will no longer scaffold
new projects with [kube-rbac-proxy][kube-rbac-proxy].
Existing users are encouraged to switch to images hosted by the project
on [quay.io](https://quay.io/repository/brancz/kube-rbac-proxy?tab=tags&tag=latest) **OR**
to adapt their projects to utilize [Network Policies][k8s-doc-networkpolicies], following the updated scaffold guidelines.

For project updates, users can manually review scaffold changes or utilize the
provided [upgrade assistance helper](https://book.kubebuilder.io/reference/rescaffold).

Communications and guidelines would be provided along with the release.

## Motivation

- **Infrastructure Reliability Concerns**: Kubebuilder’s reliance on Google's infrastructure, which may be discontinued
at their discretion, poses a risk to image availability and project reliability. [Discussion thread](https://kubernetes.slack.com/archives/CCK68P2Q2/p1711914533693319?thread_ts=1711913605.487359&cid=CCK68P2Q2) and issues: https://github.com/kubernetes/k8s.io/issues/2647 and https://github.com/kubernetes-sigs/kubebuilder/issues/3230
- **Registry Changes and Image Availability**: The transition from `gcr.io` to [registry.k8s.io][registry.k8s.io] and
the [Container Registry][container-registry-dep] deprecation implies that **all** images provided so far by Kubebuilder
[here][kb-images-repo] will unassailable by **April 22, 2025**. [More info][container-registry-dep] and [slack ETA thread][slack-eta-thread]
- **Security and Endorsement Concerns**: [kube-rbac-proxy][kube-rbac-proxy] is a process to be part of
auth-sig for an extended period, however, it is not there yet. The Kubernetes Auth SIG’s review reveals that kube-rbac-proxy
must undergo significant updates to secure an official endorsement and to be supported, highlighting pressing concerns.
You can check the ongoing process and changes required by looking at the [project issue](https://github.com/brancz/kube-rbac-proxy/issues/238)
- **Evolving User Requirements and Deprecations**: The anticipated requirement for certificate management, potentially
necessitating cert-manager, underlines Kubebuilder's aim to simplify setup and reduce third-party dependencies. [More info, see issue #3524](https://github.com/kubernetes-sigs/kubebuilder/issues/3524)
- **Aim for a Transparent and Collaborative Infrastructure**: As an open-source project, Kubebuilder strives for
a community-transparent infrastructure that allows broader contributions. This goal aligns with our initiative
to migrate Kubebuilder CLI release builds from GCP to GitHub Actions and using Go-Releaser see [here](./../build/.goreleaser.yml),
or promoting infrastructure managed under the k8s-infra umbrella.
- **Community Feedback**: Some community members preferred its removal from the default scaffolding. [Issue 3482](https://github.com/kubernetes-sigs/kubebuilder/issues/3482)
- **Enhancing Service Monitor with Proper TLS/Certificate Usage Requested by Community:** [Issue #3657](https://github.com/kubernetes-sigs/kubebuilder/issues/3657). It is achievable with [kube-rbac-proxy][kube-rbac-proxy] OR [Network Policies][k8s-doc-networkpolicies] usage within [cert-manager][cert-manager].

### Goals

- **Maximize Protection for the Metrics Endpoint without relay in third-part(s)**: Aim to provide the highest level of
protection achievable for the metrics endpoint without relying on new third-party dependencies or the need to build
and promote images from other projects.
- **Avoid Breaking Changes**: Ensure that users who generated projects with previous versions can still use the
new version with scaffold changes and adapt their project at their convenience.
- **Sustainable Project Maintenance**: Ensure all projects scaffolded by Kubebuilder can be
maintained and supported by its maintainers.
- **Independence from Google Cloud Platform**: Move away from reliance on Google Cloud Platform,
considering the potential for unilateral shutdowns.
- **Kubernetes Umbrella Compliance**: Cease the promotion or endorsement of solutions
not yet endorsed by the Kubernetes umbrella organization, mainly when used and shipped with the workload.
- **Promote Use of External Plugins**: Adhere to Kubebuilder's directive to avoid direct third-party
integrations, favouring the support of projects through the Kubebuilder API and [external plugins][external-plugins].
This approach empowers users to add or integrate solutions with the Kubebuilder scaffold on their own, ensuring that
third-party project maintainers—who are more familiar with their solutions—can maintain and update
their integrations, as implementing it following the best practices to use their project, enhancing the user experience.
External plugins should reside within third-party repository solutions and remain up-to-date as part of those changes,
aligning with their domain of responsibility.
- **Flexible Network Policy Usage**: Allow users to opt-out of the default-enabled usage of [Network Policies][k8s-doc-networkpolicies]
if they prefer another solution, plan to deploy their solution with a vendor or use a CNI that does not support NetworkPolicies.

### Non-Goals

- **Replicate kube-rbac-proxy Features or Protection Level**: It is not a goal to provide the same features
or layer of protection as [kube-rbac-proxy][kube-rbac-proxy]. Since [Network Policies][k8s-doc-networkpolicies]operate differently
and do not offer the same kind of functionality as [kube-rbac-proxy][kube-rbac-proxy], achieving identical protection levels through
[Network Policies][k8s-doc-networkpolicies]alone is not feasible.

However, incorporating NetworkPolicies, cert-manager, and/or the features introduced
in the [controller-runtime pull request #2407][cr-pr] we are mainly addressing the security concerns that
kube-rbac-proxy handles.

## Proposal

### Phase 1: Transition to network policies

The immediate action outlined in this proposal is the replacement of [kube-rbac-proxy][kube-rbac-proxy]
with Kubernetes API NetworkPolicies.

### Phase 2: Add Cert-Manager as an Optional option to be used with metrics

Looking beyond the initial phase, this proposal envisions integrating cert-manager for TLS certificate management
and exploring synergies with new features in Controller Runtime, as demonstrated in [PR #2407](https://github.com/kubernetes-sigs/controller-runtime/pull/2407).

These enhancements would introduce encrypted communication for metrics endpoints and potentially incorporate authentication mechanisms,
significantly elevating the security model employed by projects scaffolded by Kubebuilder.

- **cert-manager**: Automates the management and issuance of TLS certificates, facilitating encrypted communication and, when configured with mTLS, adding a layer of authentication.
  Currently, we leverage cert-manager when webhooks are scaffolded. So, the proposal idea would be to allow users to enable the cert-manager for the metrics such as those provided
  and required for the webhook feature. However, it MUST be optional. One of the goals of Kubebuilder is to make it easier for new users. Therefore, new users should
  not need to deal with cert-manager by default or have the need to install it to just a quick start.

That would mean, in a follow-up to the [current open PR](https://github.com/kubernetes-sigs/kubebuilder/pull/3853) to address the above `phase 1 - Transition to NetworkPolices`,
we aim to introduce a configurable Kustomize patch that will enable patching the ServiceMonitor in `config/prometheus/monitor.yaml` and certificates similar to our
existing setup for webhooks. This enhancement will ensure more flexible deployment configurations and enhance the security
features of the service monitoring components.

Currently, in the `config/default/`, we have implemented patches for cert-manager along with webhooks, as seen in
`config/default/kustomization.yaml` ([example](https://github.com/kubernetes-sigs/kubebuilder/blob/bd0876b8132ff66da12d8d8a0fdc701fde00f54b/docs/book/src/component-config-tutorial/testdata/project/config/default/kustomization.yaml#L51-L149)).
These patches handle annotations for the cert-manager CA injection across various configurations, like
ValidatingWebhookConfiguration, MutatingWebhookConfiguration, and CRDs.

For the proposed enhancements, we need to integrate similar configurations for the ServiceMonitor.
This involves the creation of a patch file named `metrics_https_patch.yaml`, which will include
configurations necessary for enabling HTTPS for the ServiceMonitor.

Here's an example of how this configuration might look:

```sh
# [METRICS WITH HTTPS] To enable the ServiceMonitor using HTTPS, uncomment the following line
# Note that for this to work, you also need to ensure that cert-manager is enabled in your project
- path: metrics_https_patch.yaml
```

This patch should apply similar changes as the current webhook patches,
targeting necessary updates in the manifest to support HTTPS communication secured by
cert-manager certificates.

Here is an example of how the `ServiceMonitor` configured to work with cert-manager might look:

```yaml
# Prometheus Monitor Service (Metrics) with cert-manager
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    control-plane: controller-manager
    app.kubernetes.io/name: project-v4
    app.kubernetes.io/managed-by: kustomize
  name: controller-manager-metrics-monitor
  namespace: system
  annotations:
    cert-manager.io/inject-ca-from: $(NAMESPACE)/controller-manager-certificate
spec:
  endpoints:
    - path: /metrics
      port: https
      scheme: https
      bearerTokenFile: /var/run/secrets/kubernetes.io/serviceaccount/token
      tlsConfig:
        # We should recommend ensure that TLS verification is not skipped in production
        insecureSkipVerify: false
        caFile: /etc/prometheus/secrets/ca.crt # CA certificate injected by cert-manager
        certFile: /etc/prometheus/secrets/tls.crt # TLS certificate injected by cert-manager
        keyFile: /etc/prometheus/secrets/tls.key # TLS private key injected by cert-manager
  selector:
    matchLabels:
      control-plane: controller-manager
```

### Phase 3: When Controller-Runtime feature is enhanced

After we have the [issue](https://github.com/kubernetes-sigs/controller-runtime/issues/2781)
addressed, and we plan to use it to protect the endpoint. See, that would mean ensuring
that we are either `handle authentication (authn), authorization (authz)`.
Examples of its implementation can be found [here](https://github.com/kubernetes-sigs/cluster-api/blob/v1.6.3/util/flags/diagnostics.go#L79-L82).

### Phase 4: When kube-rbac-proxy be accepted under the umbrella

Once kube-rbac-proxy is included in the Kubernetes umbrella,
Kubebuilder maintainers can support its integration through a [plugin](https://kubebuilder.io/plugins/plugins).
We can following up the ongoing process and changes required for the project be accepted
by looking at the [project issue](https://github.com/brancz/kube-rbac-proxy/issues/238).

This enables a seamless way to incorporate kube-rbac-proxy into Kubebuilder scaffolds,
allowing users to run:

```sh
kubebuilder init|edit --plugins="kube-rbac-proxy/v1"
```

So that the plugin could use the [plugin/util](../pkg/plugin/util) lib provide
to comment (We can add a method like the [UncommentCode](https://github.com/kubernetes-sigs/kubebuilder/blob/72586d386cfbcaecea6321a703d1d7560c521885/pkg/plugin/util/util.go#L102))
the patches in the `config/default/kustomization` and disable the default network policy used within
and [replace the code](https://github.com/kubernetes-sigs/kubebuilder/blob/72586d386cfbcaecea6321a703d1d7560c521885/pkg/plugin/util/util.go#L231)
in the `main.go` bellow with to not use the controller-runtime
feature instead.

```go
ctrlOptions := ctrl.Options{
    MetricsFilterProvider: filters.WithAuthenticationAndAuthorization,
    MetricsSecureServing:  true,
}
```

### Documentation Updates

Each phase of implementation associated with this proposal must include corresponding
updates to the documentation. This is essential to ensure end users understand
how to enable, configure, and utilize the options effectively. Documentation updates should be
completed as part of the pull request to introduce code changes.

### Proof of Concept

- **(Phase 1)NetworkPolicies:** https://github.com/kubernetes-sigs/kubebuilder/pull/3853
- Example of Controller-Runtime new feature to protect Metrics Endpoint: https://github.com/sbueringer/controller-runtime-tests/tree/master/metrics-auth

### Risks and Mitigations

#### Loss of Previously Promoted Images

The transition to the new shared infrastructure for Kubernetes SIG projects has rendered us unable to automatically build and promote images as before.
The process only works for projects under the umbrella.
However, the k8s-infra maintainers could manually transfer these images
to the new [registry.k8s.io][registry.k8s.io] as a "contingent approach".
See: [https://explore.ggcr.dev/?repo=gcr.io%2Fk8s-staging-kubebuilder%2Fkube-rbac-proxy](https://explore.ggcr.dev/?repo=registry.k8s.io%2Fkubebuilder%2Fkube-rbac-proxy)

To continue using kube-rbac-proxy, users must update their projects to reference images
from the new registry. This requires a project update and a new release,
ensuring the image references in the `config/default/manager_auth_proxy_patch.yaml` point
to a new place.

Therefore, the best approach here for those still interested in using
kube-rbac-proxy seems to direct them to the images hosted
at [quay.io](https://quay.io/repository/brancz/kube-rbac-proxy?tab=tags&tag=latest),
which are maintained by the project itself and then,
we keep those images in the registry.k8s.io as a "contingent approach".

Ensuring that these images will continue to be promoted under any infrastructure available to
Kubebuilder is not reliable or achievable for Kubebuilder maintainers. It is definitely out of our control.

#### Impact of Google Cloud Platform Kubebuilder project

Kubebuilder hasn't received any official notice regarding a shutdown of its project there so far, but there's a proactive move to transition away
from Google Cloud Platform services due to factors beyond our control. Open communication with our community is key as
we explore alternatives. It's important to note the [Container Registry Deprecation][container-registry-dep] results
in users no longer able to consume those images from the current location from **early 2025**,
emphasizing the need to shift away from dependent images as soon as possible and communicate it extensively
through mailing lists and other channels to ensure community awareness and readiness.

## Alternatives

### Replace the current images `gcr.io/kubebuilder/kube-rbac-proxy` with `registry.k8s.io/kubebuilder/kube-rbac-proxy`

The k8s-infra maintainers assist in ensuring these images will not be lost by:
- Manually adding them to [gcr.io/k8s-staging-kubebuilder/kube-rbac-proxy](https://explore.ggcr.dev/?repo=gcr.io%2Fk8s-staging-kubebuilder%2Fkube-rbac-proxy) and promoting them via [registry.k8s.io/kubebuilder/kube-rbac-proxy](https://explore.ggcr.dev/?image=registry.k8s.io%2Fkubebuilder%2Fkube-rbac-proxy:v0.16.0).

An available option would be to communicate to users to:
- a) Replace their registry from `gcr.io/k8s-staging-kubebuilder/kube-rbac-proxy` to `registry.k8s.io/kubebuilder/kube-rbac-proxy`
- b) Clearly state in the docs, Kubebuilder scaffolds, and all channels, including email communications, that kube-rbac-proxy is in the process of becoming part of Kubernetes/auth-sig but is not yet there and hence is a "not supported/secure" solution

**Cons:**
- Kubebuilder would still not be fully compliant with its goals since it would be scaffolding a third-party integration instead of properly endorsing and promoting the usage of external-plugin APIs.
- Kubebuilder would still be promoting a solution not deemed secure/safe according to the review by auth-sig maintainers.
- We would still need to manually request k8s-infra maintainers to build and promote these images in the new registry manually.
- Changes in the manager/project solution delivered in the scaffold have a critical impact. For example, in this case, users
will need to change **ALL** projects they support and ensure that their users no longer use their previously released versions.
Following this path, when kube-rbac-proxy is accepted under the Kubernetes/auth-sig, they will start to maintain and manage
their own images, which means this path will change again, and Kubebuilder maintainers have no control over ensuring that
these images will still be available and promoted for a long period.

### Retain kube-rbac-proxy as an Opt-in Feature and move it to an alpha plugin (Unsupported Feature) AND/OR use the project registry

This alternative keeps kube-rbac-proxy out of the default scaffolds, offering it as an optional plugin for users who choose
to integrate it. Clear communication will be crucial to inform users about the implications of using kube-rbac-proxy.

**Cons:**

Mainly, all cons added for the above alternative option `Replace the current images gcr.io/kubebuilder/kube-rbac-proxy`
with `registry.k8s.io/kubebuilder/kube-rbac-proxy` within the exception that we would make clear that we kubebuilder
is unable to manage those images and move the current implementation for the alpha plugin
it would maybe make the process to move it from the Kubebuilder repository to `kube-rbac-proxy` an
easier process to allow them to work with the external plugin.

However, that is a double effort for users and Kubebuilder maintainers to deal with breaking changes
resulting from achieving the ultimate go. Therefore, it would make more sense
to encourage using external-plugins API and add this option in their
repo once, then create these intermediate steps.

[kube-rbac-proxy]: https://github.com/brancz/kube-rbac-proxy
[external-plugins]: https://kubebuilder.io/plugins/external-plugins
[registry.k8s.io]: https://github.com/kubernetes/registry.k8s.io
[container-registry-dep]: https://cloud.google.com/artifact-registry/docs/transition/prepare-gcr-shutdown
[kb-images-repo]: https://console.cloud.google.com/gcr/images/kubebuilder/GLOBAL/kube-rbac-proxy
[slack-eta-thread]: https://kubernetes.slack.com/archives/CCK68P2Q2/p1712622102206909
[cr-pr]: https://github.com/kubernetes-sigs/controller-runtime/pull/2407
[k8s-doc-networkpolicies]: https://kubernetes.io/docs/concepts/services-networking/network-policies/
[cert-manager]:https://cert-manager.io/
