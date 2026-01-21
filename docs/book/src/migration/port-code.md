# Step 3: Port Custom Code

After reorganizing your project (Step 1) and executing scaffolding commands from discovery (Step 2), use AI to port your custom code to the new project.

<aside class="warning">

<h1>Important: Best Effort Only</h1>

The AI instructions below are provided as an **example** to help you get started. Due to the complexity and variety of Kubebuilder projects, we **cannot guarantee** it will work perfectly for all projects or be 100% accurate.

You should:
- Adapt the instructions to your specific use case
- **Validate ALL changes** made by AI carefully
- Be prepared to manually fix issues
- Not rely 100% on AI for correctness

The instructions may help you understand how to approach certain migration scenarios, but you remain responsible for ensuring correctness.

</aside>

<aside class="warning">

<h1>Prerequisites</h1>

Before using these AI instructions:
1. You've reorganized your project using Step 1 (`make build` succeeds)
2. You've backed up the reorganized project to `../migration-backup/`
3. You've discovered and executed all scaffolding commands from Step 2
4. `make build` succeeds in the new scaffolded project

</aside>

## Instructions to provide to your AI assistant

Copy and paste these instructions to your AI assistant:

```
Port custom code from Kubebuilder project backup to new scaffolded project.

CONTEXT:
What is scaffold vs custom:
- Scaffold: Auto-generated boilerplate by Kubebuilder (has "// TODO(user):" comments)
- Custom: Your business logic that replaces TODOs

Backup location: ../migration-backup/ (your old project with custom code)
New project: . (newly scaffolded project with TODOs to replace)

How to recognize each file type (by content, not just name):

API files (typically *_types.go):
- Have marker: // +kubebuilder:object:root=true
- Have structs: type <Name> struct with metav1.TypeMeta, metav1.ObjectMeta
- Have: <Name>Spec struct (desired state)
- Have: <Name>Status struct (observed state)
- Markers like: // +kubebuilder:validation:...

Controller files (typically *_controller.go):
- Have struct: type <Name>Reconciler struct { client.Client; Scheme *runtime.Scheme }
- Have function: func (r *<Name>Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
- Have function: func (r *<Name>Reconciler) SetupWithManager(mgr ctrl.Manager) error
- May have: // +kubebuilder:rbac markers before Reconcile

Webhook files (typically *_webhook.go):
- OLD pattern: func (r *<Name>) Default(), func (r *<Name>) ValidateCreate() error
- NEW pattern: type <Name>CustomDefaulter struct, func (d *<Name>CustomDefaulter) Default(ctx context.Context, obj *<Name>) error
- Conversion: func (*<Name>) Hub(), func (r *<Name>) ConvertTo(...), func (r *<Name>) ConvertFrom(...)

Main file:
- Has: func main()
- Has: ctrl.NewManager(...)
- Registers controllers and webhooks

File paths after Step 1:
- APIs in: api/v1/ or api/<group>/v1/
- Controllers in: internal/controller/ or internal/controller/<group>/
- Webhooks in: internal/webhook/v1/ or internal/webhook/<group>/v1/
- Main: cmd/main.go

Files to NEVER edit (auto-generated):
- config/crd/bases/*.yaml (generated from make manifests)
- config/rbac/role.yaml (generated from make manifests)
- config/webhook/manifests.yaml (generated from make manifests)
- **/zz_generated.*.go (generated from make generate)
- PROJECT file (managed by CLI)

Critical markers to NEVER remove:
- // +kubebuilder:scaffold:* (Kubebuilder injects code here)

Make command sequence:
- After editing APIs or markers: make generate && make manifests
- After editing Go code: make build
- After all changes: make lint-fix && make generate && make manifests && make all && make test

Common markers in API files:
- // +kubebuilder:validation:Required
- // +kubebuilder:validation:Minimum=1
- // +kubebuilder:validation:Pattern="^[a-z]+$"
- // +kubebuilder:printcolumn:name="Status",type=string,JSONPath=...

RBAC markers in controller files:
- // +kubebuilder:rbac:groups=<group>,resources=<resource>,verbs=get;list;watch;create;update;patch;delete
- // +kubebuilder:rbac:groups=<group>,resources=<resource>/status,verbs=get;update;patch
- // +kubebuilder:rbac:groups=<group>,resources=<resource>/finalizers,verbs=update

References:
- Kubebuilder Book: https://book.kubebuilder.io
- Markers Reference: https://book.kubebuilder.io/reference/markers.html
- controller-runtime: https://github.com/kubernetes-sigs/controller-runtime
- controller-tools: https://github.com/kubernetes-sigs/controller-tools

PORT CUSTOM CODE (in this order):

1. Port go.mod dependencies FIRST:

   Compare ../migration-backup/go.mod with current go.mod

   a. For packages in backup but NOT in new (exclude k8s.io/*, sigs.k8s.io/controller-*):
      - Run: go get <package>@<version>

   b. For packages in BOTH with different versions:
      - Keep the HIGHER (newer) version
      - If backup has newer version: go get <package>@<newer-version>
      - If new scaffold has newer version: keep it (don't downgrade)
      - NOTE: Old projects can have newer versions than scaffold

   After ALL: run go mod tidy

2. Port API type definitions:

   For each *_types.go in backup to new (paths match after Step 1):
   Backup: ../migration-backup/api/v1/<kind>_types.go
   New: api/v1/<kind>_types.go

   Port:
   - Custom fields in Spec and Status structs
   - ALL +kubebuilder markers (validation, printcolumn, resource, etc.)
   - Documentation comments
   - Custom types (enums, type aliases)
   - REMOVE "// TODO(user):" comments when adding fields

   NEVER remove: // +kubebuilder:scaffold:* or // +kubebuilder:object:root=true

   After each: go mod tidy && make generate && make manifests

3. Port controller implementations:

   For each controller (paths match after Step 1):
   Backup: ../migration-backup/internal/controller/<kind>_controller.go
   New: internal/controller/<kind>_controller.go

   Port in order:
   a. Additional imports (ADD to existing)
   b. Custom constants, variables, types, interfaces (before Reconciler struct)
   c. Custom fields in <Kind>Reconciler struct
   d. ALL +kubebuilder:rbac markers (place before Reconcile)
   e. Reconcile() body (REMOVE "// TODO(user):" and paste custom logic)
   f. ALL helper functions (closures and standalone)
   g. SetupWithManager customizations (if any beyond default .For().Named().Complete())

   After each: go mod tidy && make generate && make manifests && make build

4. Port webhooks:

   CRITICAL: Code pattern depends on controller-runtime version!

   Webhooks (paths match after Step 1):
   Backup: ../migration-backup/internal/webhook/v1/<kind>_webhook.go
   New: internal/webhook/v1/<kind>_webhook.go

   Detect pattern by reading backup file:
   - Has "func (r *<Kind>) Default() {": OLD pattern (needs adaptation)
   - Has "func (d *<Kind>CustomDefaulter) Default(ctx": NEW pattern (direct copy)

   IF OLD pattern - ADAPT:
   - Default(): Extract logic, paste after type assertion, change 'r.' to '<kind>.', add return nil, REMOVE TODO
   - Validate*(): Extract logic, paste after assertion, change 'r.' to '<kind>.', change return types, REMOVE TODO
   - Conversion: Copy Hub/ConvertTo/ConvertFrom directly (no change needed)

   IF NEW pattern - DIRECT COPY:
   - Copy CustomDefaulter/CustomValidator structs and all methods
   - Copy helper functions and imports

   After each: go mod tidy && make manifests && make build

5. Port main.go customizations:

   Backup: ../migration-backup/cmd/main.go
   New: cmd/main.go

   Compare and port ONLY custom additions:
   - Custom manager options
   - Custom command-line flags
   - Custom initialization before mgr.Start()
   - Additional scheme registrations

   DO NOT port standard scaffold (controller/webhook setup, manager config)

   After: make build

6. Port config settings (ADAPT, don't copy):

   a. config/default/kustomization.yaml - Compare and adapt:
      - Uncomment webhook/certmanager if you have webhooks
      - Update namespace/namePrefix if custom
      - Match metrics configuration
      - Add custom patches/resources
      DO NOT copy entire file

   b. Other config/*/kustomization.yaml - Check for custom patches, adapt if needed

   c. Custom config dirs - Copy any additional dirs: config/dev/, config/prod/, etc.

   After: make build-installer

7. Port config samples and customizations:
   - Sample CRs: Copy ../migration-backup/config/samples/*.yaml to config/samples/
   - Makefile: Copy custom targets from backup (preserve scaffolded targets)
   - Dockerfile: Apply custom build steps from backup

8. Port ALL tests:
   - Controller tests: Copy *_controller_test.go from backup
   - Webhook tests: Copy *_webhook_test.go (adapt if pattern changed)
   - E2E tests: Copy test/e2e/* if exist
   - Integration tests: Copy test/integration/* if exist

9. Port additional files:
   - README: Port custom sections (don't replace entire file)
   - Additional dirs: Copy docs/, scripts/, examples/, charts/, testdata/ if exist
   - Root files: Copy .env, VERSION, CHANGELOG.md, CONTRIBUTING.md if exist
   - .github workflows: Copy custom workflows

   DO NOT port: dist/, bin/, vendor/

10. Verify nothing missed:
   - Run: diff -r --brief ../migration-backup/ . | grep "Only in ../migration-backup"
   - Port any custom files found (ignore: .git/, bin/, vendor/, dist/, zz_generated.*, go.sum, auto-gen configs)
   - Verify key files have custom code (APIs, controllers, webhooks)

11. Final verification:
   - Run: go mod tidy
   - Run: make lint-fix
   - Run: make generate
   - Run: make manifests
   - Run: make build
   - Run: make build-installer
   - Run: make test

   Success: no errors, tests pass, functionally identical to backup

IMPORTANT REMINDERS:
- NEVER edit auto-generated files (already listed in CONTEXT above)
- NEVER remove // +kubebuilder:scaffold:* comments
- REMOVE "// TODO(user):" when replacing with custom code
- ADAPT config YAML files, don't copy entire files
- Port EVERYTHING except: .git/, bin/, vendor/, dist/, zz_generated.*, go.sum
- Follow make command sequence from CONTEXT above
```

## What AI Will Do

The AI will:

1. **Detect layouts** - Compare old and new project structures
2. **Port API definitions** - Custom fields, markers, documentation
3. **Port controller logic** - Imports, types, Reconcile(), helpers, RBAC, SetupWithManager
4. **Adapt webhooks** - Handle pattern changes if needed, port all logic and helpers
5. **Port main.go** - Only custom initialization, flags, and manager options
6. **Port configs** - kustomization.yaml, samples, Makefile, Dockerfile
7. **Port dependencies** - Add packages to go.mod, run go mod tidy
8. **Port tests** - Controller tests, webhook tests, e2e tests, integration tests
9. **Port additional files** - README, docs/, scripts/, .github/, any custom directories
10. **Verify completely** - Run lint-fix, generate, manifests, build, test

## After AI Completes

**Critical: Review carefully!**

<aside class="warning">

<h1>Manual Review Required</h1>

After AI ports the code:

1. **Review webhook implementations** - If migrating from v3, verify the pattern adaptation is correct
2. **Check RBAC markers** - Ensure all permissions are preserved
3. **Test custom logic** - Run `make test` and verify your business logic works
4. **Compare critical files** - Diff API types, controller logic, webhook validation
5. **Test in a cluster** - Deploy and verify actual behavior

AI can make mistakes. You are responsible for ensuring correctness.

</aside>

## Example: What Gets Ported

### API Custom Fields

**From backup** (`api/v1/captain_types.go`):
```go
type CaptainSpec struct {
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=100
    Replicas int32 `json:"replicas"`

    // +kubebuilder:validation:Pattern=`^[a-z]+$`
    Name string `json:"name"`
}
```

**To new project** (TODO removed, custom fields added):
```go
type CaptainSpec struct {
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=100
    Replicas int32 `json:"replicas"`

    // +kubebuilder:validation:Pattern=`^[a-z]+$`
    Name string `json:"name"`
}
```

### Controller Reconcile Logic

**From backup** (Reconcile function body):
```go
func (r *CaptainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    // Your custom reconciliation logic here
    var captain crewv1.Captain
    if err := r.Get(ctx, req.NamespacedName, &captain); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Custom business logic...

    return ctrl.Result{}, nil
}
```

**To new project** (TODO removed, custom logic added):
```go
func (r *CaptainReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    // Custom reconciliation logic from backup
    var captain crewv1.Captain
    if err := r.Get(ctx, req.NamespacedName, &captain); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Custom business logic...

    return ctrl.Result{}, nil
}
```

### Webhook Adaptation (v3 to v4)

**From go/v3 backup**:
```go
func (r *Captain) Default() {
    if r.Spec.Replicas == 0 {
        r.Spec.Replicas = 1
    }
}
```

**To go/v4 new project**:
```go
func (d *CaptainCustomDefaulter) Default(ctx context.Context, obj *crewv1.Captain) error {
    // Ported logic adapted (obj is type-safe, no assertion needed):
    if obj.Spec.Replicas == 0 {
        obj.Spec.Replicas = 1
    }

    return nil
}
```

## Next Steps

After AI ports your code:

1. Check if nothing is missed, broken or wrongly ported
2. Deploy to test cluster - Verify behavior

<aside class="note">

<h1>If You Have a Helm Chart</h1>

If you had a Helm chart to distribute your project, you may want to consider regenerate with the [helm/v2-alpha plugin](../plugins/available/helm-v2-alpha.md)
and then applying your customizations on top.

</aside>