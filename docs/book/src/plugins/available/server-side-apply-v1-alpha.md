# Server-Side Apply Plugin `(server-side-apply/v1-alpha)`

The `server-side-apply` plugin scaffolds APIs with controllers that use [Server-Side Apply][server-side-apply], enabling safer field management when resources are shared between your controller and users or other controllers.

By using this plugin, you will get:

- A controller implementation using Server-Side Apply patterns
- Automatic generation of apply configuration types for type-safe Server-Side Apply
- Makefile integration to generate apply configurations alongside DeepCopy methods
- Tests using the apply configuration patterns

<aside class="note">
<h1>Example</h1>

See the `project-v4-with-server-side-apply` directory under the [testdata][testdata]
directory in the Kubebuilder project to check an example
of scaffolding created using this plugin.

The `Application` API and its controller was scaffolded
using the command:

```shell
kubebuilder create api \
  --group apps \
  --version v1 \
  --kind Application \
  --plugins="server-side-apply/v1-alpha"
```
</aside>

## When to use it?

Use this plugin when:

- **Multiple controllers manage the same resource**: Your controller manages some fields while other controllers or users manage others
- **Users customize your CRs**: Users add their own labels, annotations, or spec fields that your controller shouldn't overwrite
- **Partial field management**: You only want to manage specific fields and leave others alone
- **Avoiding conflicts**: You want declarative field ownership tracking to prevent accidental overwrites

**Don't use it when:**
- Your controller is the sole owner of the resource (traditional Update/Patch is simpler)
- You manage the entire object (no shared ownership)
- Simple CRUD operations where you control everything

## How does it work?

### Traditional Update vs Server-Side Apply

**Traditional approach** (without this plugin):

```go
func (r *MyResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    var resource myv1.MyResource
    if err := r.Get(ctx, req.NamespacedName, &resource); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Problem: This overwrites ALL fields, including user customizations
    resource.Spec.Replicas = 3
    resource.Labels["managed-by"] = "my-controller"
    
    if err := r.Update(ctx, &resource); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

**Server-Side Apply approach** (with this plugin):

```go
import myv1apply "example.com/project/pkg/applyconfiguration/apps/v1"

func (r *MyResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Build desired state - only specify fields you want to manage
    resourceApply := myv1apply.MyResource(req.Name, req.Namespace).
        WithSpec(myv1apply.MyResourceSpec().
            WithReplicas(3)).
        WithLabels(map[string]string{
            "managed-by": "my-controller",
        })

    // Apply - only manages the fields you specified above
    // User's custom labels/annotations are preserved!
    if err := r.Patch(ctx, &myv1.MyResource{
        ObjectMeta: metav1.ObjectMeta{
            Name:      req.Name,
            Namespace: req.Namespace,
        },
    }, client.Apply, client.ForceOwnership, client.FieldOwner("my-controller")); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

### What gets generated

1. **Apply configuration types** in `pkg/applyconfiguration/`:
   ```
   pkg/applyconfiguration/
   └── apps/v1/
       ├── application.go
       ├── applicationspec.go
       └── applicationstatus.go
   ```

2. **Makefile target** to generate apply configurations:
   ```makefile
   APPLYCONFIGURATION_PATHS ?= ./api/apps/v1
   
   .PHONY: generate
   generate: controller-gen
       "$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt" paths="./..."
       "$(CONTROLLER_GEN)" applyconfiguration:headerFile="hack/boilerplate.go.txt" \
           paths="$(APPLYCONFIGURATION_PATHS)" \
           output:applyconfiguration:artifacts:config=pkg/applyconfiguration
   ```

3. **Controller using Server-Side Apply patterns**

## How to use it?

### 1. Initialize your project

```shell
kubebuilder init --domain example.com --repo example.com/myproject
```

### 2. Create API with the plugin

```shell
kubebuilder create api \
  --group apps \
  --version v1 \
  --kind Application \
  --plugins="server-side-apply/v1-alpha"
```

### 3. Customize your controller

The scaffolded controller includes a TODO where you specify which fields to manage:

```go
// TODO(user): Build desired state using apply configuration
resourceApply := appsv1apply.Application(req.Name, req.Namespace).
    WithSpec(appsv1apply.ApplicationSpec())
    // Add your desired fields here
```

### 4. Generate and run

```shell
make manifests generate
make test
make run
```

## Mixing with traditional APIs

You can use this plugin for **specific APIs only**. Other APIs in the same project can use traditional Update:

```shell
# API A - traditional approach (no plugin)
kubebuilder create api --group core --version v1 --kind Config

# API B - with Server-Side Apply plugin
kubebuilder create api --group apps --version v1 --kind Workload \
  --plugins="server-side-apply/v1-alpha"

# API C - traditional approach (no plugin)
kubebuilder create api --group core --version v1 --kind Status
```

**Result:**
- `Config` and `Status` controllers use traditional Update
- `Workload` controller uses Server-Side Apply
- Only `Workload` has apply configurations generated
- All make targets (`build-installer`, `test`, etc.) work unchanged

## Subcommands

The `server-side-apply` plugin includes the following subcommand:

- `create api`: Scaffolds the API with a controller using Server-Side Apply patterns

## Affected files

When using the `create api` command with this plugin, the following
files are affected:

- `api/<group>/<version>/*_types.go`: Scaffolds the API types (same as standard)
- `internal/controller/*_controller.go`: Scaffolds controller using Server-Side Apply
- `config/crd/bases/*`: Scaffolds CRD (same as standard)
- `config/samples/*`: Scaffolds sample CR (same as standard)
- `Makefile`: Adds apply configuration generation for this API
- `.gitignore`: Adds `pkg/applyconfiguration/` (first time only)
- `cmd/main.go`: Registers the controller (same as standard)

## Generated file structure

After creating an API with this plugin:

```
api/
└── apps/v1/
    ├── application_types.go
    └── zz_generated.deepcopy.go

internal/controller/
└── application_controller.go      # Uses Server-Side Apply

pkg/applyconfiguration/             # Generated by 'make generate'
└── apps/v1/
    ├── application.go
    ├── applicationspec.go
    └── applicationstatus.go

Makefile                            # Updated with applyconfiguration target
.gitignore                          # Excludes pkg/applyconfiguration/
```

## Real-world example

### Scenario: Multi-tenant application platform

You're building an operator that manages applications for different teams:

```go
// Application CRD - users customize with their labels, annotations, resources
type Application struct {
    metav1.TypeMeta
    metav1.ObjectMeta
    Spec   ApplicationSpec
    Status ApplicationStatus
}

type ApplicationSpec struct {
    Image    string
    Replicas int32
    // ... users might add custom fields
}
```

**Problem with traditional Update:**
- Your controller sets `Image` and `Replicas`
- User adds custom label: `team: platform-team`
- Your controller reconciles and overwrites, removing user's label

**Solution with Server-Side Apply:**

```go
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Only manage the fields we care about
    appApply := appsv1apply.Application(req.Name, req.Namespace).
        WithSpec(appsv1apply.ApplicationSpec().
            WithImage("nginx:latest").
            WithReplicas(3))

    // Apply - user's custom labels are preserved!
    if err := r.Patch(ctx, &appsv1.Application{
        ObjectMeta: metav1.ObjectMeta{
            Name:      req.Name,
            Namespace: req.Namespace,
        },
    }, client.Apply, client.ForceOwnership, client.FieldOwner("application-controller")); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

**Result:**
- Controller manages: `spec.image`, `spec.replicas`
- User can add: custom labels, annotations, other spec fields
- No conflicts - API server tracks field ownership

## Best practices

### 1. Always specify FieldOwner

```go
client.FieldOwner("my-controller-name")
```

This identifies your controller in the managed fields.

### 2. Use ForceOwnership carefully

```go
client.ForceOwnership  // Takes ownership even if another manager owns the field
```

Only use when you're certain your controller should own all specified fields.

### 3. Only manage what you need

```go
// Good - only manage replicas
appApply := appsv1apply.Application(name, ns).
    WithSpec(appsv1apply.ApplicationSpec().
        WithReplicas(3))

// Avoid - managing entire spec might conflict with users
appApply := appsv1apply.Application(name, ns).
    WithSpec(spec) // Don't set the entire spec
```

### 4. Handle conflicts

```go
err := r.Patch(ctx, obj, client.Apply, client.FieldOwner("my-controller"))
if err != nil {
    if errors.IsConflict(err) {
        // Conflict detected - another manager owns the same field
        // Decide: retry, log, or force ownership
    }
}
```

## Migration from traditional Update

To migrate an existing controller to use Server-Side Apply:

1. **Enable the plugin for new APIs** (keep existing APIs unchanged)
2. **Or manually convert**:
   - Run `make generate` to create apply configurations
   - Update Makefile to include your API in `APPLYCONFIGURATION_PATHS`
   - Refactor controller to use `Patch(Apply)` instead of `Update`

## Additional resources

For more details on Server-Side Apply concepts and patterns, see:

- [Server-Side Apply Reference](../../reference/server-side-apply.md) - Concepts and theory
- [Kubernetes Server-Side Apply Documentation][server-side-apply]
- [controller-gen CLI Reference](../../reference/controller-gen.md)

## Plugin Compatibility

**Cannot be used with deploy-image plugin:**

The server-side-apply and deploy-image plugins scaffold different controller implementations and cannot be used together for the same API.

```bash
# This will fail:
kubebuilder create api --group apps --version v1 --kind App \
  --image=nginx:latest \
  --plugins=deploy-image/v1-alpha,server-side-apply/v1-alpha

# Instead, choose one plugin per API:
kubebuilder create api --group apps --version v1 --kind App \
  --plugins=server-side-apply/v1-alpha  # OR deploy-image/v1-alpha
```

## Troubleshooting

### Apply configurations not generated

**Problem:** Running `make generate` doesn't create apply configuration files

**Solution:** Check that:
1. `APPLYCONFIGURATION_PATHS` is set in your Makefile  
2. Your API has the `+kubebuilder:ac:generate=true` marker in `groupversion_info.go`
3. Files are generated in `api/<group>/<version>/applyconfiguration/` (not `pkg/`)

### Conflicts on every reconcile

**Problem:** Getting conflicts when applying

**Solution:** 
- Make sure you're using the same `FieldOwner` consistently
- Check if another controller is managing the same fields
- Consider using `ForceOwnership` if you should own those fields

### Traditional APIs broken after adding plugin

**Problem:** Old APIs stop working after adding Server-Side Apply plugin

**Solution:** The plugin only affects the specific API it was used with. Check that:
- `APPLYCONFIGURATION_PATHS` only includes plugin APIs
- Traditional controllers still use `Update()` not `Patch(Apply)`

[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[server-side-apply]: https://kubernetes.io/docs/reference/using-api/server-side-apply/
[testdata]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/testdata
