# Creating your own plugins

## Overview

You can extend the Kubebuilder API to create your own plugins. If [extending the CLI][extending-cli], your plugin will be implemented in your project and registered to the CLI as has been done by the [SDK][sdk] project. See its [cli code][sdk-cli-pkg] as an example.

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

Kubebuilder and SDK are both broadly adopted projects which leverage the [controller-runtime][controller-runtime] project.  They both allow users to build solutions using the [Operator Pattern][operator-pattern] and follow common standards.

Adopting these standards can bring significant benefits, such as joining forces on maintaining the common standards as the features provided by Kubebuilder and take advantage of the contributions made by the community. This allows you to focus on the specific needs and requirements for your plugin and use-case.

And then, you will also be able to use custom plugins and options currently or in the future which might to be provided by these projects as any other which decides to persuade the same standards.

</aside>

## Custom Plugins 

Note that users are also able to use plugins to customize their scaffold and address specific needs. See that Kubebuilder provides the [declarative][declarative-code] plugin which can be used when for example an API is scaffold:

```sh
kubebuider create api [options] --plugins=go/v3,declarative/v1
``` 

This plugin will perform a custom scaffold using the [kubebuilder declarative pattern][kubebuilder-declarative-pattern].

In this way, by [Extending the Kubebuilder CLI][extending-cli], you can also create custom plugins such this one. Feel free to check its implementation in [`pkg/plugins/golang/declarative`][declarative-code].

## Future vision for Kubebuilder Plugins 

As the next steps for the plugins, its possible to highlight three initiatives so far, which are:

- [Plugin phase 2.0][plugin-2.0]: allow the Kubebuilder CLI or any other CLI, which is [Extending the Kubebuilder CLI][extending-cli], to discover external plugins, in this way, allow the users to use these external options as helpers to perform the scaffolds with the tool.   
- [Config-gen][config-gen]: the config-gen option has been provided as an alpha option in the Kubebuilder CLI(`kubebuilder alpha config-gen`) to encourage its contributions. The idea of this option would simplify the config scaffold. For further information see its [README][config-gen-readme].
- [New Plugin (`deploy-image.go.kubebuilder.io/v1beta1`) to generate code][new-plugin-gen]: its purpose is to provide an arch-type that will scaffold the APIs and Controllers with the required code to deploy and manage solutions on the cluster. 

Please, feel to contribute with them as well. Your contribution to the project is very welcome.  

[sdk-cli-pkg]: https://github.com/operator-framework/operator-sdk/blob/master/internal/cmd/operator-sdk/cli/cli.go
[sdk-ansible]: https://github.com/operator-framework/operator-sdk/tree/master/internal/plugins/ansible/v1
[sdk-helm]: https://github.com/operator-framework/operator-sdk/tree/master/internal/plugins/helm/v1
[operator-pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[plugin-2.0]: https://github.com/kubernetes-sigs/kubebuilder/issues/1378
[config-gen-readme]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/cli/alpha/config-gen/README.md
[config-gen]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/cli/alpha/config-gen
[plugins-phase1-design-doc-1.5]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1-5.md
[extending-cli]: extending-cli.md
[new-plugin-gen]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/code-generate-image-plugin.md
[kubebuilder-declarative-pattern]: https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern
[declarative-code]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/pkg/plugins/golang/declarative
[sdk]: https://github.com/operator-framework/operator-sdk