---
title: controller-observation-plugin
authors:
  - "@Kavinjsir"
reviewers:
  - "@varsha"
  - "@camila"
approvers:
creation-date: 2022-06-06
last-updated: 2022-07-04
status: provisional
see-also:
  - ""
replaces:
  - ""
superseded-by:
  - ""
---

# a-grafana-dashboard-plugin-to-observe-controller-metrics

## Release Signoff Checklist

- [ ] Enhancement is `implementable`
- [ ] Design details are appropriately documented from clear requirements
- [ ] Test plan is defined
- [ ] Graduation criteria for dev preview, tech preview, GA

## Open Questions [optional]

1. What are the ideal panels/queries(metrics) that users expect in general?
2. How would users expect to have a dashboard of rich content OR multiple dashboards for different aspects?

   - What about having a complex dashboard with comprehensive panels and some other dashboards focusing on different aspects?

3. Necessary to have dashboard(s) installed automatically? And how?

   - (e.g.: metrics are automatically exposed to prometheus.)

4. How is the plugin backwards compatible for users who have prometheus scaffolded?

   - Is it compatible only with specific versions of prometheus? Or do can we have an existing kb project and add this plugin to scaffold grafana manifests smoothly. Do we need any other changes to an old project structure.

5. What does the ideal approach for the plugin to create the grafana manifests?
   > Implement the basic approach now to let plugin scaffold raw JSON manifests. (No need from user input)
6. Should we scaffold a dir with the config to install the grafana? See: https://grafana.com/docs/grafana/latest/installation/kubernetes/
   > Probably not since users may different ways to setup their grafana services.
7. Where should the default directory be scaffold?
   > Currently use `grafana/`

## Related issues and PRs

- [Feature Request: Grafana Plugin Initial](https://github.com/kubernetes-sigs/kubebuilder/issues/2718)
- [Feature Request: Add scaffolder to render cpu&memory usage](https://github.com/kubernetes-sigs/kubebuilder/pull/2797)

## Summary

This EP will provide a way for the operator author to visualize the runtime metrics in order to observe the operator status.

We currently have controller-runtime metrics exposed by default for all KB operators at an endpoint through Prometheus.

In order to visualize those same metrics, every operator author has to individually put in the effort to create a dashboard.

This plugin intends to reduce the effort of operator authors to create dashboards and instead provides with the manifests that will allow them to integrate their controller metrics with grafana easily.

On top of the basic dashboard provided by us having default metrics, operator authors can also add custom metrics and visualize them in the format they are comfortable in.

The functionality will be implemented by a new plugin that can generate the 'manifests' to be loaded on Grafana.

In the context of this EP, the Grafana 'manifests' are mainly:

- JSON files to be directly loaded on Grafana dashboards
- TBD

## Motivation

The [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) provides rich metrics to measure controllers' status from various perspectives.

Currently, users of Kubebuilder need to design and create their own Grafana dashboards to visualize the metrics.
It would be great if we can provide a unified and comprehensive approach so that users may:

- easily visualize controller/operator metrics
- avoid duplicate creation of Grafana dashboards file
- improve the observability towards the operator, keep it safe and robust

### Goals

- Provide manifests that can display operator status for common needs.
- Add a plugin ~~or enhance the current kustomize plugin~~ that scaffolds Grafana manifests.
- Enable dashboards to display panels of user-defined metrics.

### Non-Goals

- TBD

## Proposal

This proposed implementation is to create a new plugin, `grafana`, that scaffolds the manifests which are loaded by Grafana to display dashboards for controllers' status.

Basically, the manifest can be raw json file such as `controller-dashboard.json` that can be directly applied to Grafana Web UI.

As there are many useful metrics, it might be better to observe in multiple dashboards rather than one.

Hence, we may also provide optional dashboards that focus on different aspects.

For instance: work queue, reconciliation, performance, webhook.

As a result, the layout given by the plugin can be a folder of `grafana/`:

```shell
grafana/
├── controller-runtime-metrics.json # Reconciliation and workqueue status
└── controller-resources-metrics.json # Recources usage such as CPU & Memory
```

The plugin is triggered to do the scaffolding work for the layout above when the following commands are executed:

- `kubebuilder init -–plugins=grafana.kubebuilder.io/v1-alpha`: when initializing a new project

- `kubebuilder edit -–plugins=grafana.kubebuilder.io/v1-alpha`: when adding features to an existing project

#### Dashboard Panels

The creation of the panels inside the dashboard can be referred to default [controller-runtime metrics](https://github.com/kubernetes-sigs/kubebuilder/blob/master/docs/book/src/reference/metrics-reference.md).

The overall dashboard should reflect:

1. Latency: The duration of a request for certain controllers/operators, which includes both successful ops and errors ops.
2. Traffic: The measure of how busy the controllers/operators can be.
3. Errors: Error rates, or the operations/requests that takes unexpectedly long time.
4. Saturation: The measure of the possibility of overflows such as deep work queue depth, concurrencies of reconciliations, large cpu/memory usage.

Also, the dashboard should provide filters and selectors to focus on certain objects. This can be implemented by utilizing [Grafana Query Variables](https://grafana.com/docs/grafana/latest/datasources/prometheus/#query-variable).

An example can be:
![sample-dash](https://user-images.githubusercontent.com/18136486/172537982-bb4d6a6d-5b9b-4231-8d8c-860a8255bf8d.png)

### User Stories

#### Story 1

As an opeartor author, I want to create a new operator by kubebuilder.
When running the operator, I wish I can observe its status easily.

`kubebuilder init -–plugins=grafana.kubebuilder.io/v1-alpha` can initialize a new opeartor project with the dashboard manifests available. The author can directly load the manifest inside `grafana/` to have Grafana dashboard installed.

#### Story 2

As an opeartor author, I have an existing operator scaffolded by kubebuilder.
And I wish I can observe its status easily.

`kubebuilder edit -–plugins=grafana.kubebuilder.io/v1-alpha` can provide addons of the dashboard manifests available.

### Implementation Details/Notes/Constraints [optional]

#### Phase 1

We will begin with the initialization of an optional plugin, which should start at `alpha` version.

Hence, the implementation should be placed at `kubebuilder/pkg/plugins/optional/grafana/v1alpha/`.

To maintain a good coding pattern, the structure of the code for the implementation should be
in consistent with [other plugins](https://github.com/kubernetes-sigs/kubebuilder/tree/master/pkg/plugins/golang/declarative/v1).

In this early stage, the plugin will assume the default controller-runtime metrics are exposed.

Once triggered, the plugin will scaffold a `grafana/` directory that contains the Grafana manifest. For instance, `controller-runtime-metrics.json`.

The operator author should be able to directly load the content of the manifest in his/her Grafana Web UI.

So the prerequisites to make the plugin available are:

- Using controller-runtime OR having the same default metrics exported
- Enabling Prometheus
- Having the metrics exposed in the /metrics endpoint

#### Phase 2 (experimental)

It's common that users have their own defined metrics for certain use cases. Custom metrics are introduced in [docs](https://master.book.kubebuilder.io/reference/metrics.html?highlight=metrics#publishing-additional-metrics).
It can be a nice try to let the plugin also scaffold out the dashboard to visualize these user-defined metrics.
A new `grafana/custom_metrics` directory can be placed:

```shell
grafana/custom_metrics/
├── dashboard.json # The manifest to display custom metrics
└── config.yaml # Entry of user input to define the dashboard
```

Initially, the `config.yaml` is empty.
Users can add values in `config.yaml` to tell the `custom metrics` with the `type` of the panel to display in the dashboard.
Then, when running `kubebuilder edit -–plugins=grafana.kubebuilder.io/v1-alpha`, the plugin will read `config.yaml` and generate `dashboard.json`.

#### Phase 3 (TBD)

If the approach and the manifest are welcomed by the community/users, it maybe good to provide more manifests.
The plugin can be enhanced to scaffold additional manifests according to user preference.
(Say, a `--mode [simple | extended] flag`)

Once triggered, the layout can be:

```shell
grafana/
├── controller-dashboard.json # The main dashboard raw json file for the simplest usage(copy/paste)
├── options
│  ├── performance.json       # CPU/Memory usaage, restful api performance
│  ├── reconcilier.json       # Reconciliation instruments
│  ├── webhook.jon            # Webhook Ops
│  └── workqueue.json         # Work queue status
└── README.md                 # Docs for the usage
```

#### Phase 4 (TBD)

This can be an alternative of **Phase 2** if multiple dashboards are considered not necessary or in a lower priority.

Since metrics exposure and visualization are closely relative, the plugin can be extended to scaffoled the prometheus manifests.
That way, this plugin won't have dependencies on other plugins, which makes it more flexible.

A simple approach maybe:
Provide the `service-monitor` that `kustomize` does.

There are two challenges:

1. How should the plugin behave when it is used with/without `kustomize` being triggered? Would there be any duplications or coverings on the same path of the layout? Or if it maybe some contridictions when using both of them?
2. The current metrics are exposed with restriction so that when the user want to query these metrics on his/her Grafana Dashboard, will it be possible to have authentication/authorization issue?

#### Phase 5 (TBD)

This is optional and is necessary to be determined by the community.

Manual installation of Grafana dashboards maybe inconvenient. In some cases, it is possible to make the process automatically when deploying the operator.

However, this may heavily depends on users' stacks of using Grafana:

1. For [prometheus-operator](https://prometheus-operator.dev/docs/developing-prometheus-rules-and-grafana-dashboards/), it supports K8s ConfigMap for dashboards installation.
2. For grafana/jsonnet, users may use [different stack base](https://github.com/grafana/jsonnet-libs/blob/master/grafana/example/tanka/environments/default/dashboards/nyc.json).
3. For Grafana Cloud, users may copy raw json to the ui.

To handle it generally, we may consider [Grafana HTTP API](https://grafana.com/docs/grafana/latest/http_api/dashboard/#create--update-dashboard).

### Risks and Mitigations

TBD

## Design Details

### Test Plan

TBD

### Graduation Criteria

TBD
