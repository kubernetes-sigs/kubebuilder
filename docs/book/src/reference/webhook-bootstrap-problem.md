# Webhook Bootstrap Problem

## The Problem

When you create a webhook for a **core Kubernetes type** (Pod, Deployment, Job, etc.), the webhook can block its own controller Pod from starting, causing a deployment deadlock.

**Example command:**
```bash
kubebuilder create webhook --group core --version v1 --kind Pod --programmatic-validation
```

**Example scenario:**

1. You create a validating webhook for Pods
2. You deploy your controller (which runs in a Pod)
3. Kubernetes tries to create your controller Pod
4. Your webhook intercepts this Pod creation
5. The webhook server isn't ready yet (it's inside the Pod being created)
6. The Pod creation hangs waiting for webhook validation
7. The webhook never starts because the Pod is blocked

**Result:** Deadlock. Your deployment fails.

## When Does This Occur?

### Core Kubernetes Types

The bootstrap problem occurs when creating webhooks for built-in Kubernetes resources:

- `core` group: Pod, Service, Namespace, ConfigMap, Secret
- `apps` group: Deployment, StatefulSet, DaemonSet, ReplicaSet
- `batch` group: Job, CronJob
- Other built-in types

**Why?** Your webhook validates the same type of resource that your controller deployment creates (typically Pods or Deployments).

### Custom CRDs

The bootstrap problem **does not occur** with custom resource webhooks:

- Your webhook validates `MyResource` objects
- Your controller runs as a Pod
- Pods and MyResources are different types
- No circular dependency

## How to Fix

Configure your webhook to **skip validating its own resources** using either `namespaceSelector` or `objectSelector`.

### Option 1: namespaceSelector (Recommended)

Exclude the entire namespace where your webhook runs.

**Step 1:** Add label to the Namespace in `config/manager/manager.yaml`:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  labels:
    control-plane: controller-manager
    app.kubernetes.io/name: my-project
    app.kubernetes.io/managed-by: kustomize
    webhook-excluded: "true"
  name: system
```

**Step 2:** Create patch file `config/webhook/namespaceselector_patch.yaml`:

```yaml
# For mutating webhooks (--defaulting)
- op: add
  path: /webhooks/0/namespaceSelector
  value:
    matchExpressions:
    - key: webhook-excluded
      operator: DoesNotExist
```

For validating webhooks (`--programmatic-validation`), create a similar patch targeting `ValidatingWebhookConfiguration`.

**Step 3:** Add patch to `config/webhook/kustomization.yaml`:

```yaml
resources:
- manifests.yaml
- service.yaml

patches:
- path: namespaceselector_patch.yaml
  target:
    group: admissionregistration.k8s.io
    version: v1
    kind: MutatingWebhookConfiguration
    name: mutating-webhook-configuration
```

**Step 4:** Deploy:

```bash
make deploy IMG=<your-image>
```

### Option 2: objectSelector

Exclude specific labeled Pods from webhook validation.

**Step 1:** Add label to Pods in `config/manager/manager.yaml`:

```yaml
spec:
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: manager
      labels:
        control-plane: controller-manager
        app.kubernetes.io/name: my-project
        webhook-excluded: "true"
```

**Step 2-4:** Same as Option 1, but use `objectSelector` instead of `namespaceSelector` in the patch file.

### Multiple Webhooks

If you created webhooks for multiple core types (e.g., Pod and Deployment), you'll have multiple webhook entries.

**Check webhook count:**

```bash
make manifests
grep "  name: m" config/webhook/manifests.yaml  # Count mutating webhooks
grep "  name: v" config/webhook/manifests.yaml  # Count validating webhooks
```

**Example output:**

```
  name: mpod-v1.kb.io         # Index 0
  name: mdeployment-v1.kb.io  # Index 1
```

**Add patches for all indices** in your patch file:

```yaml
- op: add
  path: /webhooks/0/namespaceSelector
  value:
    matchExpressions:
    - key: webhook-excluded
      operator: DoesNotExist
- op: add
  path: /webhooks/1/namespaceSelector
  value:
    matchExpressions:
    - key: webhook-excluded
      operator: DoesNotExist
```

### Mixed Webhooks (CRD + Core Types)

If you have both custom CRD webhooks and core type webhooks:

- CRD webhooks appear first in the configuration
- Core type webhooks appear after
- Count **all** webhooks and add patches for the indices of your core type webhooks

**Example:** If you have 1 CRD webhook (index 0) and 1 core type webhook (index 1), your patch should target index 1:

```yaml
- op: add
  path: /webhooks/1/namespaceSelector
  value:
    matchExpressions:
    - key: webhook-excluded
      operator: DoesNotExist
```

## Choosing Between namespaceSelector and objectSelector

| Feature | namespaceSelector | objectSelector |
|---------|-------------------|----------------|
| Excludes | Entire namespace | Specific pods |
| Scope | Broad | Fine-grained |
| Best for | Dedicated webhook namespace | Shared namespace |
| Complexity | Simple | More targeted |

**Recommendation:** Use `namespaceSelector` unless you need fine-grained control.

## References

- [Kubernetes Admission Webhook Best Practices](https://kubernetes.io/docs/concepts/cluster-administration/admission-webhooks-good-practices/)
- [namespaceSelector API Reference](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#matching-requests-namespaceselector)
- [objectSelector API Reference](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#matching-requests-objectselector)
