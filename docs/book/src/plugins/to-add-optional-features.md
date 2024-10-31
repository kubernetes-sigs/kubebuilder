## To add optional features

The following plugins are useful to generate code and take advantage of optional features

| Plugin                                            | Key                     | Description                                                                                                                                         |
|---------------------------------------------------|-------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------|
| [grafana.kubebuilder.io/v1-alpha][grafana]        | `grafana/v1-alpha`      | Optional helper plugin which can be used to scaffold Grafana Manifests Dashboards for the default metrics which are exported by controller-runtime. |
| [deploy-image.go.kubebuilder.io/v1-alpha][deploy] | `deploy-image/v1-alpha` | Optional helper plugin which can be used to scaffold APIs and controller with code implementation to Deploy and Manage an Operand(image).           |
| [helm.kubebuilder.io/v1-alpha][helm]              | `helm/v1-alpha`         | Optional helper plugin which can be used to scaffold a Helm Chart to distribute the project under the `dist` directory                              |

[grafana]: ./available/grafana-v1-alpha.md
[deploy]: ./available/deploy-image-plugin-v1-alpha.md
[helm]: ./available/helm-v1-alpha.md