# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load("@io_bazel_rules_go//go:def.bzl", "GoPath", "go_context", "go_path", "go_rule")

def _compute_genrule_variables(resolved_srcs, resolved_outs):
    variables = {
        "SRCS": cmd_helper.join_paths(" ", resolved_srcs),
        "OUTS": cmd_helper.join_paths(" ", resolved_outs),
    }
    if len(resolved_srcs) == 1:
        variables["<"] = list(resolved_srcs)[0].path
    if len(resolved_outs) == 1:
        variables["@"] = list(resolved_outs)[0].path
    return variables

def _go_genrule_impl(ctx):
    go = go_context(ctx)

    all_srcs = depset(go.stdlib.files)
    label_dict = {}
    go_paths = []

    for dep in ctx.attr.srcs:
        all_srcs += dep.files
        label_dict[dep.label] = dep.files

    for go_path in ctx.attr.go_paths:
        all_srcs += go_path.files
        label_dict[go_path.label] = go_path.files

        gp = go_path[GoPath]
        ext = gp.gopath_file.extension
        if ext == "":
            # mode is 'copy' - path is just the gopath
            go_paths.append(gp.gopath_file.path)
        elif ext == "tag":
            # mode is 'link' - path is a tag file in the gopath
            go_paths.append(gp.gopath_file.dirname)
        else:
            fail("Unknown extension on gopath file: '%s'." % ext)

    cmd = [
        "set -e",
        "export GOPATH=" + ctx.configuration.host_path_separator.join(["$$(pwd)/" + p for p in go_paths]),
        ctx.attr.cmd.strip(" \t\n\r"),
    ]
    resolved_inputs, argv, runfiles_manifests = ctx.resolve_command(
        command = "\n".join(cmd),
        attribute = "cmd",
        expand_locations = True,
        make_variables = _compute_genrule_variables(all_srcs, depset(ctx.outputs.outs)),
        tools = ctx.attr.tools,
        label_dict = label_dict,
    )

    paths = [go.root + "/bin", "/bin", "/usr/bin"]
    ctx.action(
        inputs = list(all_srcs) + resolved_inputs,
        outputs = ctx.outputs.outs,
        env = ctx.configuration.default_shell_env + go.env + {
            "PATH": ctx.configuration.host_path_separator.join(paths),
        },
        command = argv,
        progress_message = "%s %s" % (ctx.attr.message, ctx),
        mnemonic = "GoGenrule",
    )

# We have codegen procedures that depend on the "go/*" stdlib packages
# and thus depend on executing with a valid GOROOT. _go_genrule handles
# dependencies on the Go toolchain and environment variables; the
# macro go_genrule handles setting up GOPATH dependencies (using go_path).
_go_genrule = go_rule(
    _go_genrule_impl,
    attrs = {
        "srcs": attr.label_list(allow_files = True),
        "tools": attr.label_list(
            cfg = "host",
            allow_files = True,
        ),
        "outs": attr.output_list(mandatory = True),
        "cmd": attr.string(mandatory = True),
        "go_paths": attr.label_list(),
        "importpath": attr.string(),
        "message": attr.string(),
        "executable": attr.bool(default = False),
    },
    output_to_genfiles = True,
)

# Genrule wrapper for tools which need dependencies in a valid GOPATH
# and access to the Go standard library and toolchain.
#
# Go source dependencies specified through the go_deps argument
# are passed to the rules_go go_path rule to build a GOPATH
# for the provided genrule command.
#
# The command can access the generated GOPATH through the GOPATH
# environment variable.
def go_genrule(name, go_deps, **kw):
    go_path_name = "%s~gopath" % name
    go_path(
        name = go_path_name,
        mode = "link",
        visibility = ["//visibility:private"],
        deps = go_deps,
    )
    _go_genrule(
        name = name,
        go_paths = [":" + go_path_name],
        **kw
    )
