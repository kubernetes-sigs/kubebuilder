# Changing things up

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

## Setting up an exising project

If you've previously generated a project we can add support for parsing the
config file by making the following changes to `main.go`.

First, add a new `flag` to specify the path that the component config file
should be loaded from.

```go
var configFile string
flag.StringVar(&configFile, "config", "",
    "The controller will load its initial configuration from this file. "+
        "Omit this flag to use the default configuration values. "+
        "Command-line flags override configuration from this file."
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

Lastly, we'll change the `NewManager` call to use the `options` varible we
defined above.

```go
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
```

With that out of the way, we can get on to defining our new config!

[tutorial-source]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/component-config-tutorial/testdata/project
