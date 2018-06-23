load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["buildtar.go"],
    importpath = "k8s.io/repo-infra/tools/build_tar",
    visibility = ["//visibility:private"],
    deps = [
        "//vendor/github.com/golang/glog:go_default_library",
        "//vendor/golang.org/x/build/pargzip:go_default_library",
    ],
)

go_binary(
    name = "build_tar",
    embed = [":go_default_library"],
    visibility = ["//visibility:public"],
)
