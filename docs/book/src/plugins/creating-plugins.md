# Creating your own plugins

[extending-cli]: extending-cli.md
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[operator-pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator
[sdk-ansible]: https://sdk.operatorframework.io/docs/building-operators/ansible/
[sdk-cli-pkg]: https://pkg.go.dev/github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/cli
[sdk-helm]: https://sdk.operatorframework.io/docs/building-operators/helm/
[sdk]: https://github.com/operator-framework/operator-sdk

## Overview

You can extend the Kubebuilder API to create your own plugins. If [extending the CLI][extending-cli], your plugin will be implemented in your project and registered to the CLI as has been done by the [SDK][sdk] project. See its [CLI code][sdk-cli-pkg] as an example.

## Language-based Plugins

Kubebuilder offers the Golang-based operator plugins, which will help its CLI tool users create projects following the [Operator Pattern][operator-pattern].

The [SDK][sdk] project, for example, has language plugins for [Ansible][sdk-ansible] and [Helm][sdk-helm], which are similar options but for users who would like to work with these respective languages and stacks instead of Golang.

Note that Kubebuilder provides the `kustomize.common.kubebuilder.io` to help in these efforts. This plugin will scaffold the common base without any specific language scaffold file to allow you to extend the Kubebuilder style for your plugins.

In this way, currently, you can [Extend the CLI][extending-cli] and use the `Bundle Plugin` to create your language plugins such as:

```go
  mylanguagev1Bundle, _ := plugin.NewBundle(language.DefaultNameQualifier, plugin.Version{Number: 1},
		kustomizecommonv1.Plugin{}, // extend the common base from Kubebuilder
		mylanguagev1.Plugin{}, // your plugin language which will do the scaffolds for the specific language on top of the common base
	)
```

If you do not want to develop your plugin using Golang, you can follow its standard by using the binary as follows:

```sh
kubebuilder init --plugins=kustomize
```

Then you can, for example, create your implementations for the sub-commands `create api` and `create webhook` using your language of preference.

<aside class="note">
<h1>Why use the Kubebuilder style?</h1>

Kubebuilder and SDK are both broadly adopted projects which leverage the [controller-runtime][controller-runtime] project. They both allow users to build solutions using the [Operator Pattern][operator-pattern] and follow common standards.

Adopting these standards can bring significant benefits, such as joining forces on maintaining the common standards as the features provided by Kubebuilder and take advantage of the contributions made by the community. This allows you to focus on the specific needs and requirements for your plugin and use-case.

And then, you will also be able to use custom plugins and options currently or in the future which might to be provided by these projects as any other which decides to persuade the same standards.

</aside>

## Custom Plugins

Note that users are also able to use plugins to customize their scaffolds and address specific needs.

See that Kubebuilder provides the [`deploy-image`][deploy-image] plugin that allows the user to create the controller & CRs which will deploy and manage an image on the cluster:

```sh
kubebuilder create api --group example.com --version v1alpha1 --kind Memcached --image=memcached:1.6.15-alpine --image-container-command="memcached,-m=64,modern,-v" --image-container-port="11211" --run-as-user="1001" --plugins="deploy-image/v1-alpha"
```

This plugin will perform a custom scaffold following the [Operator Pattern][operator-pattern].

Another example is the [`grafana`][grafana] plugin that scaffolds a new folder container manifests to visualize operator status on Grafana Web UI:

```sh
kubebuilder edit --plugins="grafana.kubebuilder.io/v1-alpha"
```

In this way, by [Extending the Kubebuilder CLI][extending-cli], you can also create custom plugins such this one.

[grafana]: https://github.com/kubernetes-sigs/kubebuilder/tree/v3.7.0/pkg/plugins/optional/grafana/v1alpha
[deploy-image]: https://github.com/kubernetes-sigs/kubebuilder/tree/v3.7.0/pkg/plugins/golang/deploy-image/v1alpha1
