# Multiple controllers per resource

Kubebuilder supports multiple named controllers for the same API resource. This allows different reconciliation logic for the same resource type.

## Usage

### Creating the first controller

```bash
kubebuilder create api --group crew --version v1 --kind Captain \
  --resource=true \
  --controller=true
```

Creates:

- API types in `api/v1/captain_types.go`
- Controller in `internal/controller/captain_controller.go` with struct `CaptainReconciler`
- Registration in `cmd/main.go`

Kubebuilder records the controller under its default name, the lowercase kind:

```yaml
resources:
- api:
    crdVersion: v1
  controllers:
  - name: captain
  group: crew
  kind: Captain
  version: v1
```

Pass `--controller-name` if you want a different name for this first controller.

### Adding additional controllers

```bash
kubebuilder create api --group crew --version v1 --kind Captain \
  --resource=false \
  --controller=true \
  --controller-name=captain-backup
```

Creates:

- Controller in `internal/controller/captain_backup_controller.go` with struct `CaptainBackupReconciler`
- Additional registration in `cmd/main.go`

The API is only created once. Additional controllers reference the existing API.

```yaml
resources:
- api:
    crdVersion: v1
  controllers:
  - name: captain
  - name: captain-backup
  group: crew
  kind: Captain
  version: v1
```

## Controller naming

### Storage

Controller names are stored in the PROJECT file exactly as given to `--controller-name`.
When the flag is omitted, the lowercase kind is stored.

### Code generation

Names are normalized for Go code:

- **File name**: Replace hyphens with underscores: `captain-backup` → `captain_backup_controller.go`
- **Struct name**: Convert to PascalCase and append Reconciler: `captain-backup` → `CaptainBackupReconciler`
- **Runtime name**: Use exact name from PROJECT: `Named("captain-backup")`
- **Multigroup**: Prefix with group name: `Named("crew-captain-backup")`

The default controller name preserves the kind's capitalization in the struct name:
`mykind` for `MyKind` generates `MyKindReconciler`.

### Validation rules

1. Names must be unique within a resource
2. Names must be DNS-1035 labels: start with a lowercase letter, end with a lowercase letter or digit, contain only lowercase letters, digits, and hyphens, and have at most 63 characters
3. Different names that generate the same reconciler struct are rejected (e.g., `captain-backup` and `captain--backup`)

Reconciler struct names must also be unique across resources in the same controller
package. Multigroup projects use a separate controller package for each group.
To create a controller for another API version of the same kind, use
`--controller-name` to choose a different name.

## Controller coordination

Multiple controllers for the same resource require coordination to avoid conflicts:

- **Field ownership**: Each controller should manage different fields
- **Finalizers**: Use unique names: `{controller-name}.example.com/finalizer`
- **Status updates**: Assign different status subfields to each controller
- **Conditional logic**: Use labels or annotations to route resources to specific controllers

Kubebuilder scaffolds the controllers but does not manage coordination between them.

## Migration from the legacy format

Existing PROJECT files with `controller: true` continue to work without
`--controller-name`. The next command that saves the PROJECT file converts each
legacy controller to a `controllers` entry named after the lowercase kind.
The migration preserves the controller's file name and reconciler struct name.

<aside class="note" role="note">
<p class="note-title">If a PROJECT file carries both formats</p>

If a resource contains both `controller: true` and `controllers:`, Kubebuilder adds
the legacy controller's default name to the list. If that name is already listed,
Kubebuilder keeps a single entry.

If two names generate the same reconciler struct, Kubebuilder reports the collision
and leaves the PROJECT file unchanged. Choose names that generate distinct
reconcilers before saving the file again.
</aside>

## Common errors

**"duplicate controller name"**: Two controllers have the same name. Use unique names.

**"conflicts with ... both generate"**: Different names generate the same reconciler struct name. Choose distinct names.

**"controller with name ... already exists"**: Controller already exists for this resource. Use a different name.

**"resource already has controllers defined"**: An unnamed request with
`--resource=false` cannot select a controller when the resource has several
controllers or only a non-default controller. Pass `--controller-name`, or use
`--controller=false` to skip controller scaffolding.
