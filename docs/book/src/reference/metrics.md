# Metrics

By default, controller-runtime builds a global prometheus registry and
publishes [a collection of performance metrics](/reference/metrics-reference.md) for each controller.

<aside class="note warning">
<h1>IMPORTANT: If you are using `kube-rbac-proxy`</h1>

**Images provided under `gcr.io/kubebuilder/` will be unavailable from March 18, 2025.**

- **Projects initialized with Kubebuilder versions `v3.14` or lower** utilize [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) to protect the metrics endpoint. Therefore, you might want to continue using kube-rbac-proxy by simply replacing the image or changing how the metrics endpoint is protected in your project.

- **However, projects initialized with Kubebuilder versions `v4.1.0` or higher** have a similar protection using authn/authz enabled by default via Controller-Runtime's feature [WithAuthenticationAndAuthorization](https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/metrics/filters#WithAuthenticationAndAuthorization).
In this case, you might want to upgrade your project or simply ensure that you have applied the same code changes to it.

> Please ensure that you update your configurations accordingly to avoid any disruptions.

### ❓ Why is this happening?

Kubebuilder has been rebuilding and re-tagging these images for several years. However, due to recent infrastructure changes for projects under the Kubernetes umbrella, we now require the use of shared infrastructure. But as [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) is in a process to be a part of it, but not yet, sadly we cannot build and promote these images using the new k8s infrastructure. To follow up the ongoing process and changes required for the project be accepted by, see: https://github.com/brancz/kube-rbac-proxy/issues/238

Moreover, Google Cloud Platform has [deprecated the Container Registry](https://cloud.google.com/artifact-registry/docs/transition/transition-from-gcr), which has been used to promote these images.

Additionally, ongoing changes and the phase-out of the previous GCP infrastructure mean that **Kubebuilder maintainers are no longer able to support, build, or ensure the promotion of these images.** For further information, please check the proposal for this change and its motivations [here](https://github.com/kubernetes-sigs/kubebuilder/pull/2345).

### How the metrics endpoint can be protected ?

- **(Protection enabled by default from release `v4.1.0`)** By using Controller-Runtime's feature [WithAuthenticationAndAuthorization](https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/metrics/filters#WithAuthenticationAndAuthorization) which can handle `authn/authz` similar what was provided via `kube-rbac-proxy`.
- By using NetworkPolicies. ([example](https://github.com/prometheus-operator/kube-prometheus/discussions/1907#discussioncomment-3896712))
- By integrating cert-manager with your metrics service you can secure the endpoint via TLS encryption
- **(Not support or promoted by Kubebuilder)** By still using [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) and the image provided by the project ([quay.io/brancz/kube-rbac-proxy](https://quay.io/repository/brancz/kube-rbac-proxy)) or from any other source

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
    image: curlimages/curl:7.78.0
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
<aside class="warning">
<h1>Changes Recommended for Production</h1>

The default scaffold in `cmd/main.go` uses a **controller-runtime feature** to
automatically generate a self-signed certificate to secure the metrics server.
While this is convenient for development and testing, it is not recommended
for production.

You can mount a certificate into the Manager Deployment and configure the
metrics server to use it, as shown below:

```go
if secureMetrics {
	...

    // Specify the path where the certificate is mounted
    metricsServerOptions.CertDir = "/tmp/k8s-metrics-server/metrics-certs"
    metricsServerOptions.CertName = "tls.crt"
    metricsServerOptions.KeyName = "tls.key"
}
```

Additionally, review the configuration file at `config/prometheus/monitor.yaml`
to ensure secure integration with Prometheus. **If `insecureSkipVerify: true` is
enabled, certificate verification is turned off. This is not recommended for production**
as it exposes the system to man-in-the-middle attacks, potentially allowing
unauthorized access to metrics data.

</aside>


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

### By exposing the metrics endpoint using HTTPS and Cert-Manager

Integrating `cert-manager` with your metrics service enables secure
HTTPS access via TLS encryption. Follow the steps below to configure
your project to expose the metrics endpoint using HTTPS with cert-manager.

1. **Enable Cert-Manager in `config/default/kustomization.yaml`:**
    - Uncomment the cert-manager resource to include it in your project:

      ```yaml
      - ../certmanager
      ```

2. **Enable the Patch for the `ServiceMonitor` to Use the Cert-Manager-Managed Secret `config/prometheus/kustomization.yaml`:**
    - Add or uncomment the `ServiceMonitor` patch to securely reference the cert-manager-managed secret, replacing insecure configurations with secure certificate verification:

      ```yaml
      - path: monitor_tls_patch.yaml
        target:
          kind: ServiceMonitor
      ```

3. **Enable the Patch to Mount the Cert-Manager-Managed Secret in the Controller Deployment in `config/default/kustomization.yaml`:**
    - Use the `manager_webhook_patch.yaml` (or create a custom metrics patch) to mount the `serving-cert` secret in the Manager Deployment.

      ```yaml
      - path: manager_webhook_patch.yaml
      ```

4. **Update `cmd/main.go` to Use the Certificate Managed by Cert-Manager:**
    - Modify `cmd/main.go` to configure the metrics server to use the cert-manager-managed certificates.
   Uncomment the lines for `CertDir`, `CertName`, and `KeyName`:

      ```go
      if secureMetrics {
		...
        metricsServerOptions.CertDir = "/tmp/k8s-metrics-server/metrics-certs"
		metricsServerOptions.CertName = "tls.crt"
		metricsServerOptions.KeyName = "tls.key"
      }
      ```

### By using Network Policy (You can optionally enable)

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