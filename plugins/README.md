# Kubebuilder plugins

**Status: Experimental**

We are developing a plugin system to kubebuilder, so that we can generate
operators that follow other patterns.

While plugins remain experimental, you must pass the `KUBEBUILDER_ENABLE_PLUGINS=1`
environment variable to enable plugin functionality.  (Any non-empty
value will work!)

When you specify `KUBEBUILDER_ENABLE_PLUGINS=1`, a flag `--pattern` will become
available for resource generation.  Specifying `--pattern=addon` will change
resource code generation to generate code that follows the addon pattern, as
being developed in the
[addon-operators](https://github.com/kubernetes-sigs/addon-operators)
subproject.

The `pattern=addon` plugin is intended to serve both as an example of a plugin,
and as a real-world use case for driving development of the plugin system.  We
don't intend for the plugin system to become an emacs competitor, but it must be
sufficiently flexible to support the various patterns of operators that
kubebuilder will generate.

## Plugin model

We intend for plugins to be packaged in a separate binary, which will be
executed by the `kubebuilder` main binary.  Data will be piped to the binary via
stdin, and returned over stdout.  The serialization format will likely either be
yaml or json (to be determined!).

While we are developing this functionality though, we are developing it using an
in-process golang interface named `Plugin`, defined in
[pkg/scaffold/scaffold.go](../pkg/scaffold/scaffold.go).  The interface is a
simple single-method interface that is intended to mirror the data-in / data-out
approach that will be used when executing a plugin in a separate binary.  When
we have more stability of the plugin, we intend to replace the in-process
implementation with a implementation that `exec`s a plugin in a separate binary.

The approach being prototyped is that we pass a model of the full state of the
generation world to the Plugin, which returns the full state of the generation
world after making appropriate changes.  We are starting to define a `model`
package which includes a `Universe` comprising the various `File`s that are
being generated, along with the inputs like the `Boilerplate` and the `Resource`
we are currently generating.  A plugin can change the `Contents` of `File`s, or
add/remove `File`s entirely.
