## To add optional features

The following plugins are useful to generate code and take advantage of optional features

| Plugin                                              | Key                     | Description                                                                                                                                                                           |
|-----------------------------------------------------|-------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| [autoupdate.kubebuilder.io/v1-alpha][autoupdate]    | `autoupdate/v1-alpha`   | Optional helper which scaffolds a scheduled worker that helps keep your project updated with changes in the ecosystem, significantly reducing the burden of manual maintenance. |
| [deploy-image.go.kubebuilder.io/v1-alpha][deploy]   | `deploy-image/v1-alpha` | Optional helper plugin which can be used to scaffold APIs and controller with code implementation to Deploy and Manage an Operand(image).                                             |
| [grafana.kubebuilder.io/v1-alpha][grafana]          | `grafana/v1-alpha`      | Optional helper plugin which can be used to scaffold Grafana Manifests Dashboards for the default metrics which are exported by controller-runtime.                                   |
| [helm.kubebuilder.io/v1-alpha][helm-v1alpha] (deprecated) | `helm/v1-alpha`         | **Deprecated** - Optional helper plugin which can be used to scaffold a Helm Chart to distribute the project under the `dist` directory. Use v2-alpha instead.                     |
| [helm.kubebuilder.io/v2-alpha][helm-v2alpha]        | `helm/v2-alpha`         | Optional helper plugin which dynamically generates Helm charts from kustomize output, preserving all customizations                                                                     |
| [server-side-apply.go.kubebuilder.io/v1-alpha][server-side-apply] | `server-side-apply/v1-alpha` | Optional helper plugin which scaffolds APIs with controllers using Server-Side Apply for safer field management when resources are shared between controllers and users.    |

[grafana]: ./available/grafana-v1-alpha.md
[deploy]: ./available/deploy-image-plugin-v1-alpha.md
[helm-v1alpha]: ./available/helm-v1-alpha.md
[helm-v2alpha]: ./available/helm-v2-alpha.md
[autoupdate]: ./available/autoupdate-v1-alpha.md
[server-side-apply]: ./available/server-side-apply-v1-alpha.md