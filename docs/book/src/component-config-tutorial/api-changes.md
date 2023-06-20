# Changing things up

<aside class="note warning">
<h1>Component Config is deprecated</h1>

The ComponentConfig has been deprecated in the Controller-Runtime since its version 0.15.0.  [More info](https://github.com/kubernetes-sigs/controller-runtime/issues/895)
Moreover, it has undergone breaking changes and is no longer functioning as intended.
As a result, Kubebuilder, which heavily relies on the Controller Runtime, has also deprecated this feature,
no longer guaranteeing its functionality from version 3.11.0 onwards. You can find additional details on this issue [here](https://github.com/kubernetes-sigs/controller-runtime/issues/2370).

Please, be aware that it will force Kubebuilder remove this option soon in future release.

</aside>

This tutorial will show you how to create a custom configuration file for your
project by modifying a project generated with the `--component-config` flag
passed to the `init` command. The full tutorial's source can be found 
[here][tutorial-source]. Make sure you've gone through the [installation 
steps](/quick-start.md#installation) before continuing.

## New project:

```bash
# we'll use a domain of tutorial.kubebuilder.io,
# so all API groups will be <group>.tutorial.kubebuilder.io.
kubebuilder init --domain tutorial.kubebuilder.io --component-config
```

## Setting up an existing project

If you've previously generated a project we can add support for parsing the
config file by making the following changes to `main.go`.

First, add a new `flag` to specify the path that the component config file
should be loaded from.

```go
var configFile string
flag.StringVar(&configFile, "config", "",
    "The controller will load its initial configuration from this file. "+
        "Omit this flag to use the default configuration values. "+
            "Command-line flags override configuration from this file.")
```

Now, we can setup the `Options` struct and check if the `configFile` is set,
this allows backwards compatibility, if it's set we'll then use the `AndFrom`
function on `Options` to parse and populate the `Options` from the config.


```go
var err error
options := ctrl.Options{Scheme: scheme}
if configFile != "" {
    options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile))
    if err != nil {
        setupLog.Error(err, "unable to load the config file")
        os.Exit(1)
    }
}
```

<aside class="note warning">

<h1>Your Options may have defaults from flags.</h1>

If you have previously allowed other `flags` like `--metrics-bind-addr` or 
`--enable-leader-election`, you'll want to set those on the `Options` before
loading the config from the file.

</aside>

Lastly, we'll change the `NewManager` call to use the `options` variable we
defined above.

```go
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
```

With that out of the way, we can get on to defining our new config!

[tutorial-source]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/component-config-tutorial/testdata/project

Create the file `/config/manager/controller_manager_config.yaml` with the following content:

```yaml
apiVersion: controller-runtime.sigs.k8s.io/v1alpha1
kind: ControllerManagerConfig
health:
  healthProbeBindAddress: :8081
metrics:
  bindAddress: 127.0.0.1:8080
webhook:
  port: 9443
leaderElection:
  leaderElect: true
  resourceName: ecaf1259.tutorial.kubebuilder.io
# leaderElectionReleaseOnCancel defines if the leader should step down volume
# when the Manager ends. This requires the binary to immediately end when the
# Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
# speeds up voluntary leader transitions as the new leader don't have to wait
# LeaseDuration time first.
# In the default scaffold provided, the program ends immediately after
# the manager stops, so would be fine to enable this option. However,
# if you are doing or is intended to do any operation such as perform cleanups
# after the manager stops then its usage might be unsafe.
# leaderElectionReleaseOnCancel: true
```

Update the file `/config/manager/kustomization.yaml` by adding at the bottom the following content:

```yaml
generatorOptions:
  disableNameSuffixHash: true

configMapGenerator:
- name: manager-config
  files:
  - controller_manager_config.yaml
```

Update the file `default/kustomization.yaml` by adding under the [`patchesStrategicMerge:` key](https://kubectl.docs.kubernetes.io/references/kustomize/builtins/#_patchesstrategicmerge_) the following patch:

```yaml
patchesStrategicMerge:
# Mount the controller config file for loading manager configurations
# through a ComponentConfig type
- manager_config_patch.yaml
```

Update the file `default/manager_config_patch.yaml` by adding under the `spec:` key the following patch:

```yaml
spec:
  template:
    spec:
      containers:
      - name: manager
        args:
        - "--config=controller_manager_config.yaml"
        volumeMounts:
        - name: manager-config
          mountPath: /controller_manager_config.yaml
          subPath: controller_manager_config.yaml
      volumes:
      - name: manager-config
        configMap:
          name: manager-config
```
