# License Header

## What is it?

Kubebuilder creates a file at `hack/boilerplate.go.txt` that contains your license header. This header is automatically added to all generated Go files.

The boilerplate file is used by:
- Kubebuilder when creating new files (`create api`, `create webhook`, etc.)
- `controller-gen` when generating code (DeepCopy methods, etc.)
- `make generate` and `make manifests` commands

By default, Kubebuilder uses the Apache 2.0 license.

<aside class="warning" role="note">
<p class="note-title">Boilerplate is a Template</p>

`hack/boilerplate.go.txt` is a **template** used when generating or regenerating files. Changing this file does **not** automatically update existing files.

**To apply changes to existing files:**
- Run `make generate && make manifests` to update generated code
- Run `kubebuilder alpha generate` to regenerate the entire project
- Or create new files - they'll use the updated template

</aside>

## How to Customize

### Option 1: Use Built-in Licenses

Use the `--license` flag during initialization or to update an existing project:

```bash
# During initialization
kubebuilder init --domain example.com --license apache2 --owner "Your Name"

# After initialization (update existing project)
kubebuilder edit --license apache2 --owner "Your Company"
```

Available license values:
- `apache2` - Apache 2.0 License (default)
- `copyright` - Copyright notice only (no license text)
- `none` - No license header

### Option 2: Use a Custom License File

Provide your own license header from a file. Kubebuilder will read the content from your file and copy it to `hack/boilerplate.go.txt`:

```bash
# During initialization
kubebuilder init --domain example.com --license-file ./my-header.txt

# After initialization
kubebuilder edit --license-file ./my-header.txt
```

**How it works:**
1. Kubebuilder reads your custom license file (can be any path)
2. Copies the content to `hack/boilerplate.go.txt`

<aside class="note" role="note">
<p class="note-title">License File Override</p>

The `--license-file` flag overrides the `--license` flag. If you provide `--license-file`, a boilerplate will always be created, even if you also specify `--license none`.

</aside>

Use this when you need:
- A license not available in the built-in templates
- A specific company license format
- Custom copyright notices

### Option 3: Edit the File Directly

After initialization, you can edit `hack/boilerplate.go.txt` directly:

```bash
vim hack/boilerplate.go.txt
```

## Custom License File Format

Your custom license file must include Go comment delimiters (`/*` and `*/`).

**Example** (`my-license-header.txt`):

```go
/*
Copyright YEAR Your Company Name.

Licensed under the MIT License.
See LICENSE file in the project root for full license text.
*/
```

<aside class="warning" role="note">
<p class="note-title">Automatic Year Updates</p>

Use `YEAR` in your boilerplate and it will be automatically replaced with the current year:

```go
/*
Copyright YEAR Your Company.
*/
```

**How it works:**
- **Kubebuilder** replaces `YEAR` when creating files (`create api`, `create webhook`, etc.)
- **Controller-gen** replaces `YEAR` when generating code (`make generate`)

The Makefile passes the year to controller-gen:

```makefile
YEAR ?= $(shell date +%Y)

.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt",year=$(YEAR) paths="./..."
```

Both automatically use the current year, so your boilerplate stays up-to-date.

</aside>

## Applying License Changes to Existing Files

After updating `hack/boilerplate.go.txt`, choose how to apply the changes:

**Regenerate Generated Code (Recommended):**
```bash
make generate  # Update DeepCopy methods
make manifests # Update CRDs, RBAC, webhooks
```
This updates generated code but keeps your hand-written files unchanged.

**Regenerate Entire Project:**
```bash
kubebuilder alpha generate
```
<aside class="warning" role="note">
<p class="note-title">Commit First</p>

This overwrites all scaffolded files. Commit your changes before running.

</aside>

**New Files Only:**
New files will automatically use the updated template:
```bash
kubebuilder create api --group myapp --version v1 --kind MyNewKind
```

**Manual Updates:**
For hand-written files, manually update headers using `hack/boilerplate.go.txt` as a reference.

## How It's Used

`hack/boilerplate.go.txt` is used when generating or regenerating files:

- `kubebuilder create api` and `create webhook` - new files
- `make generate` - DeepCopy methods, etc.
- `make manifests` - CRDs, RBAC, webhooks
- `kubebuilder alpha generate` - entire project

All tools automatically reference `hack/boilerplate.go.txt`. The Makefile is already configured - no changes needed.

## Examples

### Example 1: Apache 2.0 License (Default)

Use the built-in Apache 2.0 license:

```bash
kubebuilder init --domain example.com --license apache2 --owner "Your Company"
```

This generates:

```go
/*
Copyright YEAR Your Company.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
```

### Example 2: Copyright Only

Use the built-in `copyright` license for just a copyright notice:

```bash
kubebuilder init --domain example.com --license copyright --owner "Your Company"
```

This generates:

```go
/*
Copyright YEAR Your Company.
*/
```

### Example 3: Custom License (MIT)

For licenses not built-in, create a custom file `mit-header.txt`:

```go
/*
Copyright YEAR Your Name.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
```

Then initialize your project:

```bash
kubebuilder init --domain example.com --license-file ./mit-header.txt
```

**What happens:**
1. Kubebuilder reads the content from `./mit-header.txt`
2. Creates `hack/boilerplate.go.txt` with that content
3. `hack/boilerplate.go.txt` becomes the source of truth for all generated files

## Related

- [Good Practices](./good-practices.md) - Best practices for project structure
- [Generating CRDs](./generating-crd.md) - How CRDs are generated with headers
- [controller-gen CLI](./controller-gen.md) - Details on code generation
