# Metrics

By default, controller-runtime builds a global prometheus registry and
publishes [a collection of performance metrics](/reference/metrics-reference.md) for each controller.

<aside class="note warning">
<h1>IMPORTANT: If you are using `kube-rbac-proxy`</h1>

**Images provided under `gcr.io/kubebuilder/` will be unavailable from March 18, 2025.**

**Projects initialized with Kubebuilder versions `v3.14` or lower** utilize [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) to protect the metrics endpoint. Therefore, you might want to continue using kube-rbac-proxy by simply replacing the image or changing how the metrics endpoint is protected in your project.

**However, projects initialized with Kubebuilder versions `v4.1.0` or higher** have a similar protection using authn/authz enabled by default via Controller-Runtime's feature [WithAuthenticationAndAuthorization](https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/metrics/filters#WithAuthenticationAndAuthorization).
In this case, you might want to upgrade your project or simply ensure that you have applied the same code changes to it.

- Check the usage in the file `config/default/manager_auth_proxy_patch.yaml` where the [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) container is patched. ([example](https://github.com/kubernetes-sigs/kubebuilder/blob/94a5ab8e52cf416a11428b15ef0f40e4aabbc6ab/testdata/project-v4/config/default/manager_auth_proxy_patch.yaml#L11-L23))
- See the file `/config/default/kustomization.yaml` where the [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) was patched by default previously. ([example](https://github.com/kubernetes-sigs/kubebuilder/blob/94a5ab8e52cf416a11428b15ef0f40e4aabbc6ab/testdata/project-v4/config/default/kustomization.yaml#L29-L33))

> Please ensure that you update your configurations accordingly to avoid any disruptions.

### If you are using OR wish to continue using [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy):

In this case, you must replace the image `gcr.io/kubebuilder/kube-rbac-proxy` for the image provided by the kube-rbac-proxy maintainers ([quay.io/brancz/kube-rbac-proxy](https://quay.io/repository/brancz/kube-rbac-proxy)), which is **not support or promoted by Kubebuilder**, or from any other registry/source that please you.

### ❓ Why is this happening?

Kubebuilder has been rebuilding and re-tagging these images for several years. However, due to recent infrastructure changes for projects under the Kubernetes umbrella, we now require the use of shared infrastructure. But as [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) is in a process to be a part of it, but not yet, sadly we cannot build and promote these images using the new k8s infrastructure. To follow up the ongoing process and changes required for the project be accepted by, see: https://github.com/brancz/kube-rbac-proxy/issues/238

Moreover, Google Cloud Platform has [deprecated the Container Registry](https://cloud.google.com/artifact-registry/docs/transition/transition-from-gcr), which has been used to promote these images.

Additionally, ongoing changes and the phase-out of the previous GCP infrastructure mean that **Kubebuilder maintainers are no longer able to support, build, or ensure the promotion of these images.** For further information, please check the proposal for this change and its motivations [here](https://github.com/kubernetes-sigs/kubebuilder/pull/2345).

### How the metrics endpoint can be protected ?

- **(Protection enabled by default from release `v4.1.0`)** By using Controller-Runtime's feature [WithAuthenticationAndAuthorization](https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.4/pkg/metrics/filters#WithAuthenticationAndAuthorization) which can handle `authn/authz` similar what was provided via `kube-rbac-proxy`.
- By still using [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) and the image provided by the project ([quay.io/brancz/kube-rbac-proxy](https://quay.io/repository/brancz/kube-rbac-proxy)) or from any other source - _(**Not support or promoted by Kubebuilder**)_
- By using NetworkPolicies. ([example](https://github.com/prometheus-operator/kube-prometheus/discussions/1907#discussioncomment-3896712))
- By integrating cert-manager with your metrics service you can secure the endpoint via TLS encryption

Further information can be found bellow in this document.

> Note that we plan use the above options to protect the metrics endpoint in the Kubebuilder scaffold in the future. For further information, please check the [proposal](https://github.com/kubernetes-sigs/kubebuilder/pull/2345).

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
Metrics: metricsserver.Options{
   ...
   FilterProvider: filters.WithAuthenticationAndAuthorization,
   ...
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

The default scaffold to configure the metrics server in `cmd/main.go` uses `TLSOpts` that rely on self-signed certificates
(SelfCerts), which are generated automatically. However, self-signed certificates are **not** recommended for production
environments as they do not offer the same level of trust and security as certificates issued by a trusted
Certificate Authority (CA).

While self-signed certificates are convenient for development and testing, they are unsuitable for production
because they do not establish a chain of trust, making them vulnerable to security threats.

Furthermore, check the configuration file located at `config/prometheus/monitor.yaml` to
ensure secure integration with Prometheus. If the `insecureSkipVerify: true` option is enabled,
it means that certificate verification is turned off. This is **not** recommended for production as
it poses a significant security risk by making the system vulnerable to man-in-the-middle attacks,
where an attacker could intercept and manipulate the communication between Prometheus and the monitored services.
This could lead to unauthorized access to metrics data, compromising the integrity and confidentiality of the information.

**In both cases, the primary risk is potentially allowing unauthorized access to sensitive metrics data.**

### Recommended Actions for a Secure Production Setup

1. **Replace Self-Signed Certificates:**
   - Instead of using `TLSOpts`, configure the `CertDir`, `CertName`, and `KeyName` options to use your own certificates.
   This ensures that your server communicates using trusted and secure certificates.

2. **Configure Prometheus Monitoring Securely:**
   - Check and update your Prometheus configuration file (`config/prometheus/monitor.yaml`) to ensure secure settings.
   - Replace `insecureSkipVerify: true` with the following secure options:

     ```yaml
     caFile: The path to the CA certificate file, e.g., /etc/metrics-certs/ca.crt.
     certFile: The path to the client certificate file, e.g., /etc/metrics-certs/tls.crt.
     keyFile: The path to the client key file, e.g., /etc/metrics-certs/tls.key.
     ```

   These settings ensure encrypted and authenticated communication between Prometheus and the monitored services, providing a secure monitoring setup.
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


### By using Network Policy

NetworkPolicy acts as a basic firewall for pods within a Kubernetes cluster, controlling traffic
flow at the IP address or port level. However, it doesn't handle authentication (authn), authorization (authz),
or encryption directly like [kube-rbac-proxy](https://github.com/brancz/kube-rbac-proxy) solution.

### By exposing the metrics endpoint using HTTPS and CertManager

Integrating `cert-manager` with your metrics service can secure the endpoint via TLS encryption.

To modify your project setup to expose metrics using HTTPS with
the help of cert-manager, you'll need to change the configuration of both
the `Service` under `config/default/metrics_service.yaml` and
the `ServiceMonitor` under `config/prometheus/monitor.yaml` to use a secure HTTPS port
and ensure the necessary certificate is applied.

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