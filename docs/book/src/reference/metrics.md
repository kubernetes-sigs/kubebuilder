# Metrics

By default, controller-runtime builds a global Prometheus registry and
publishes [a collection of performance metrics](/reference/metrics-reference.md) for each controller.


<aside class="note warning">
<h1>IMPORTANT: If you are using `kube-rbac-proxy`</h1>

Please stop using the image `gcr.io/kubebuilder/kube-rbac-proxy` as soon as possible.
Your projects will be affected and may fail to work if the image cannot be pulled.

**Images provided under `gcr.io/kubebuilder/` will be unavailable from early 2025.**

- **Projects initialized with Kubebuilder versions `v3.14` or lower** utilize [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) to protect the metrics endpoint.
In this case, you might want to upgrade your project to the latest release or ensure that you have applied the same or similar code changes.

- **However, projects initialized with Kubebuilder versions `v4.1.0` or higher** have similar protection using `authn/authz`
enabled by default via Controller-Runtime's feature [WithAuthenticationAndAuthorization](https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/metrics/filters#WithAuthenticationAndAuthorization).

If you want to continue using [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) then you MUST change
your project to use the image from another source.

> For further information, see: [kubebuilder/discussions/3907](https://github.com/kubernetes-sigs/kubebuilder/discussions/3907)

</aside>


## Metrics Configuration

By looking at the file `config/default/kustomization.yaml` you can
check the metrics are exposed by default:

```yaml
# [METRICS] Expose the controller manager metrics service.
- metrics_service.yaml
```

```yaml
patches:
   # [METRICS] The following patch will enable the metrics endpoint using HTTPS and the port :8443.
   # More info: https://book.kubebuilder.io/reference/metrics
   - path: manager_metrics_patch.yaml
     target:
        kind: Deployment
```

Then, you can check in the `cmd/main.go` where metrics server
is configured:

```go
// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
// For more info: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/metrics/server
Metrics: metricsserver.Options{
   ...
},
```

## Metrics Protection

Unprotected metrics endpoints can expose valuable data to unauthorized users,
such as system performance, application behavior, and potentially confidential
operational metrics. This exposure can lead to security vulnerabilities
where an attacker could gain insights into the system's operation
and exploit weaknesses.

### By using authn/authz (Enabled by default)

To mitigate these risks, Kubebuilder projects utilize authentication (authn) and authorization (authz) to protect the
metrics endpoint. This approach ensures that only authorized users and service accounts can access sensitive metrics
data, enhancing the overall security of the system.

In the past, the [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) was employed to provide this protection.
However, its usage has been discontinued in recent versions. Since the release of `v4.1.0`, projects have had the
metrics endpoint enabled and protected by default using the [WithAuthenticationAndAuthorization](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/metrics/server)
feature provided by controller-runtime.

Therefore, you will find the following configuration:

- In the `cmd/main.go`:

```go
if secureMetrics {
  ...
  metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
}
```

This configuration leverages the FilterProvider to enforce authentication and authorization on the metrics endpoint.
By using this method, you ensure that the endpoint is accessible only to those with the appropriate permissions.

- In the `config/rbac/kustomization.yaml`:

```yaml
# The following RBAC configurations are used to protect
# the metrics endpoint with authn/authz. These configurations
# ensure that only authorized users and service accounts
# can access the metrics endpoint.
- metrics_auth_role.yaml
- metrics_auth_role_binding.yaml
- metrics_reader_role.yaml
```

In this way, only Pods using the `ServiceAccount` token are authorized to read the metrics endpoint. For example:

```ymal
apiVersion: v1
kind: Pod
metadata:
  name: metrics-consumer
  namespace: system
spec:
  # Use the scaffolded service account name to allow authn/authz
  serviceAccountName: controller-manager
  containers:
  - name: metrics-consumer
    image: curlimages/curl:latest
    command: ["/bin/sh"]
    args:
      - "-c"
      - >
        while true;
        do
          # Note here that we are passing the token obtained from the ServiceAccount to curl the metrics endpoint
          curl -s -k -H "Authorization: Bearer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)"
          https://controller-manager-metrics-service.system.svc.cluster.local:8443/metrics;
          sleep 60;
        done
```

### **(Recommended)** Enabling certificates for Production (Disabled by default)

<aside class="warning">
<h1>Why Is This Not Enabled by Default?</h1>

This option is not enabled by default because it introduces a dependency on CertManager.
To keep the project as lightweight and beginner-friendly as possible, it is disabled by default.

</aside>

<aside class="warning">
<h1>Recommended for Production</h1>

The default scaffold in `cmd/main.go` uses a **controller-runtime feature** to
automatically generate a self-signed certificate to secure the metrics server.
While this is convenient for development and testing, it is **not** recommended
for production.

Those certificates are used to secure the transport layer (TLS).
The token authentication using `authn/authz`, which is enabled by default serves
as the application-level credential. However, for example, when you enable
the integration of your metrics with Prometheus, those certificates can be used
to secure the communication.

</aside>

Projects built with Kubebuilder releases `4.4.0` and above have the logic scaffolded
to enable the usage of certificates managed by [CertManager](https://cert-manager.io/)
for securing the metrics server. Following the steps below, you can configure your
project to use certificates managed by CertManager.

1. **Enable Cert-Manager in `config/default/kustomization.yaml`:**
    - Uncomment the cert-manager resource to include it in your project:

      ```yaml
      - ../certmanager
      ```

2. **Enable the Patch to configure the usage of the certs in the Controller Deployment in `config/default/kustomization.yaml`:**
    - Uncomment the `cert_metrics_manager_patch.yaml` to mount the `serving-cert` secret in the Manager Deployment.

      ```yaml
      # Uncomment the patches line if you enable Metrics and CertManager
      # [METRICS-WITH-CERTS] To enable metrics protected with certManager, uncomment the following line.
      # This patch will protect the metrics with certManager self-signed certs.
      - path: cert_metrics_manager_patch.yaml
        target:
          kind: Deployment
      ```
3. **Enable the CertManager replaces for the Metrics Server certificates in `config/default/kustomization.yaml`:**
    - Uncomment the replacements block bellow. It is required to properly set the DNS names for the certificates configured under `config/certmanager`.

      ```yaml
      # [CERTMANAGER] To enable cert-manager, uncomment all sections with 'CERTMANAGER' prefix.
      # Uncomment the following replacements to add the cert-manager CA injection annotations
      #replacements:
      # - source: # Uncomment the following block to enable certificates for metrics
      #     kind: Service
      #     version: v1
      #     name: controller-manager-metrics-service
      #     fieldPath: metadata.name
      #   targets:
      #     - select:
      #         kind: Certificate
      #         group: cert-manager.io
      #         version: v1
      #         name: metrics-certs
      #       fieldPaths:
      #         - spec.dnsNames.0
      #         - spec.dnsNames.1
      #       options:
      #         delimiter: '.'
      #         index: 0
      #         create: true
      #
      # - source:
      #     kind: Service
      #     version: v1
      #     name: controller-manager-metrics-service
      #     fieldPath: metadata.namespace
      #   targets:
      #     - select:
      #         kind: Certificate
      #         group: cert-manager.io
      #         version: v1
      #         name: metrics-certs
      #       fieldPaths:
      #         - spec.dnsNames.0
      #         - spec.dnsNames.1
      #       options:
      #         delimiter: '.'
      #         index: 1
      #         create: true
      #
      ```

4. **Enable the Patch for the `ServiceMonitor` to Use the Cert-Manager-Managed Secret `config/prometheus/kustomization.yaml`:**
    - Add or uncomment the `ServiceMonitor` patch to securely reference the cert-manager-managed secret, replacing insecure configurations with secure certificate verification:

      ```yaml
      # [PROMETHEUS-WITH-CERTS] The following patch configures the ServiceMonitor in ../prometheus
      # to securely reference certificates created and managed by cert-manager.
      # Additionally, ensure that you uncomment the [METRICS WITH CERTMANAGER] patch under config/default/kustomization.yaml
      # to mount the "metrics-server-cert" secret in the Manager Deployment.
      patches:
        - path: monitor_tls_patch.yaml
          target:
            kind: ServiceMonitor
      ```

    > **NOTE** that the `ServiceMonitor` patch above will ensure that if you enable the Prometheus integration,
    it will securely reference the certificates created and managed by CertManager. But it will **not** enable the
    integration with Prometheus. To enable the integration with Prometheus, you need uncomment the `#- ../certmanager`
    in the `config/default/kustomization.yaml`. For more information, see [Exporting Metrics for Prometheus](#exporting-metrics-for-prometheus).

### **(Optional)** By using Network Policy (Disabled by default)

NetworkPolicy acts as a basic firewall for pods within a Kubernetes cluster, controlling traffic
flow at the IP address or port level. However, it doesn't handle `authn/authz`.

Uncomment the following line in the `config/default/kustomization.yaml`:

```
# [NETWORK POLICY] Protect the /metrics endpoint and Webhook Server with NetworkPolicy.
# Only Pod(s) running a namespace labeled with 'metrics: enabled' will be able to gather the metrics.
# Only CR(s) which uses webhooks and applied on namespaces labeled 'webhooks: enabled' will be able to work properly.
#- ../network-policy
```

## Exporting Metrics for Prometheus

Follow the steps below to export the metrics using the Prometheus Operator:

1. Install Prometheus and Prometheus Operator.
   We recommend using [kube-prometheus](https://github.com/coreos/kube-prometheus#installing)
   in production if you don't have your own monitoring system.
   If you are just experimenting, you can only install Prometheus and Prometheus Operator.

2. Uncomment the line `- ../prometheus` in the `config/default/kustomization.yaml`.
   It creates the `ServiceMonitor` resource which enables exporting the metrics.

```yaml
# [PROMETHEUS] To enable prometheus monitor, uncomment all sections with 'PROMETHEUS'.
- ../prometheus
```

Note that, when you install your project in the cluster, it will create the
`ServiceMonitor` to export the metrics. To check the ServiceMonitor,
run `kubectl get ServiceMonitor -n <project>-system`. See an example:

```
$ kubectl get ServiceMonitor -n monitor-system
NAME                                         AGE
monitor-controller-manager-metrics-monitor   2m8s
```

<aside class="warning">
<h2>If you are using Prometheus Operator ensure that you have the required
permissions</h2>

If you are using Prometheus Operator, be aware that, by default, its RBAC
rules are only enabled for the `default` and `kube-system namespaces`. See its
guide to know [how to configure kube-prometheus to monitor other namespaces using the `.jsonnet` file](https://github.com/prometheus-operator/kube-prometheus/blob/main/docs/monitoring-other-namespaces.md).

Alternatively, you can give the Prometheus Operator permissions to monitor other namespaces using RBAC. See the Prometheus Operator
[Enable RBAC rules for Prometheus pods](https://github.com/prometheus-operator/prometheus-operator/blob/main/Documentation/user-guides/getting-started.md#enable-rbac-rules-for-prometheus-pods)
documentation to know how to enable the permissions on the namespace where the
`ServiceMonitor` and manager exist.
</aside>

Also, notice that the metrics are exported by default through port `8443`. In this way,
you are able to check the Prometheus metrics in its dashboard. To verify it, search
for the metrics exported from the namespace where the project is running
`{namespace="<project>-system"}`. See an example:

<img width="1680" alt="Screenshot 2019-10-02 at 13 07 13" src="https://user-images.githubusercontent.com/7708031/66042888-a497da80-e515-11e9-9d77-d8a9fc1159a5.png">

## Publishing Additional Metrics

If you wish to publish additional metrics from your controllers, this
can be easily achieved by using the global registry from
`controller-runtime/pkg/metrics`.

One way to achieve this is to declare your collectors as global variables and then register them using `init()` in the controller's package.

For example:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
    goobers = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "goobers_total",
            Help: "Number of goobers proccessed",
        },
    )
    gooberFailures = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "goober_failures_total",
            Help: "Number of failed goobers",
        },
    )
)

func init() {
    // Register custom metrics with the global prometheus registry
    metrics.Registry.MustRegister(goobers, gooberFailures)
}
```

You may then record metrics to those collectors from any part of your
reconcile loop. These metrics can be evaluated from anywhere in the operator code.

<aside class="note">
<h1>Enabling metrics in Prometheus UI</h1>

In order to publish metrics and view them on the Prometheus UI, the Prometheus instance would have to be configured to select the Service Monitor instance based on its labels.

</aside>

Those metrics will be available for prometheus or
other openmetrics systems to scrape.

![Screen Shot 2021-06-14 at 10 15 59 AM](https://user-images.githubusercontent.com/37827279/121932262-8843cd80-ccf9-11eb-9c8e-98d0eda80169.png)

<aside class="note">
<h1>Controller-Runtime Auth/Authz Feature Current Known Limitations and Considerations</h1>

Some known limitations and considerations have been identified. The settings for `cache TTL`, `anonymous access`, and
`timeouts` are currently hardcoded, which may lead to performance and security concerns due to the inability to
fine-tune these parameters. Additionally, the current implementation lacks support for configurations like
`alwaysAllow` for critical paths (e.g., `/healthz`) and `alwaysAllowGroups` (e.g., `system:masters`), potentially
causing operational challenges. Furthermore, the system heavily relies on stable connectivity to the `kube-apiserver`,
making it vulnerable to metrics outages during network instability. This can result in the loss of crucial metrics data,
particularly during critical periods when monitoring and diagnosing issues in real-time is essential.

An [issue](https://github.com/kubernetes-sigs/controller-runtime/issues/2781) has been opened to
enhance the controller-runtime and address these considerations.
</aside>
