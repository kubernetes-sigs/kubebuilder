package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_binary",
    "go_library",
)

go_binary(
    name = "kazel",
    embed = [":go_default_library"],
    tags = ["automanaged"],
)

go_library(
    name = "go_default_library",
    srcs = [
        "config.go",
        "diff.go",
        "generator.go",
        "kazel.go",
        "sourcerer.go",
    ],
    importpath = "k8s.io/repo-infra/kazel",
    tags = ["automanaged"],
    deps = [
        "//vendor/github.com/bazelbuild/buildtools/build:go_default_library",
        "//vendor/github.com/golang/glog:go_default_library",
    ],
)
