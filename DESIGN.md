# Kubebuilder Design Principles

This lays out some of the guiding design principles behind the Kubebuilder
project and its various components.

## Overarching

* **Libraries Over Code Generation**: Generated code is messy to maintain,
  hard for humans to change and understand, and hard to update.  Library
  code is easy to update (just increase your dependency version), easier
  to version using existing mechanisms, and more concise.

* **Copy-pasting is bad**: Copy-pasted code suffers from similar problems
  as code generation, except more acutely.  Copy-pasted code is nearly
  impossible to easy update, and frequently suffers from bugs and
  misunderstandings.  If something is being copy-pasted, it should
  refactored into a library component or remote
  [kustomize](https://sigs.k8s.io/kustomize) base.

* **Common Cases Should Be Easy**: The 80-90% common cases should be
  simple and easy for users to understand.

* **Uncommon Cases Should Be Possible**: There shouldn't be situations
  where it's downright impossible to do something within
  controller-runtime or controller-tools. It may take extra digging or
  coding, and it may involve interoperating with lower-level components,
  but it should be possible without unreasonable friction.

## Kubebuilder

* **Kubebuilder Has Opinions**: Kubebuilder exists as an opinionated
  project generator.  It should strive to give users a reasonable project
  layout that's simple enough to understand when getting started, but
  provides room to grow.  It might not match everyone's opinions, but it
  should strive to be useful to most.

* **Batteries Included**: Kubebuilder projects should contain enough
  deployment information to reasonably develop and run the scaffolded
  project.  This includes testing, deployment files, and development
  infrastructure to go from code to running containers.

## controller-tools and controller-runtime

* **Sufficient But Composable**: controller-tools and controller-runtime
  should be sufficient for building a custom controller by hand.  While
  scaffolding and additional libraries may make life easier, building
  without should be as painless as possible.  That being said, they should
  strive to be usable as building blocks for higher-level libraries as
  well.

* **Self-Sufficient Docs**: controller-tools and controller-runtime should
  strive to have self-sufficient docs (i.e. documentation that doesn't
  require reading other libraries' documentation for common use cases).
  Examples should be plentiful.

* **Contained Arcana**: Developers should not need to be experts in
  Kubernetes API machinery to develop controllers, but those familiar with
  Kubernetes API machinery should not feel out of place.  Abstractions
  should be intuitive to new users but feel familiar to experienced ones.
  Abstractions should embrace the concepts of Kubernetes (e.g. declarative
  idempotent reconcilers) while simplifying the details.

## controller-runtime

* **Abstractions Should Be Layered**: Abstractions should be built on top
  of lower layers, such that advanced users can write custom logic while
  still working within the existing model.  For instance, the controller
  builder is built on top of the event, source, and handler helpers, which
  are in turn built for use with the event, source, and handler
  interfaces.

* **Repetitive Stress Injuries Are Bad**:
  When possible, commonly used pieces should be exposed in a way that
  enables clear, concise code.  This includes aliasing groups of
  functionality under "alias" or "prelude" packages to avoid having 40
  lines of imports, including common idioms as flexible helpers, and
  infering resource information from the user's object types in client
  code.

* **A Little Bit of Magic Goes a Long Way**: In absence of generics,
  reflection is acceptable, especially when it leads to clearer, conciser
  code.  However, when possible interfaces that use reflection should be
  designed to avoid requiring the end-developer to use type assertions,
  string splitting, which are error-prone and repetitive.  These should be
  dealt with inside controller-runtime internals.

* **Defaults Over Constructors**: When not a huge performance impact,
  favor auto-defaulting and `Options` structs over constructors.
  Constructors quickly become unclear due to lack of names associated
  with values, and don't work well with optional values.

## Development

* **Words Are Better Than Letters**: Don't abbreviate variable names
  unless it's blindingly obvious what they are (e.g. `ctx` for `Context`).
  Single- and double-letter method receivers are acceptable, but single-
  and double-letter variables quickly become confusing the longer a code
  block gets.

* **Well-commented code**: Code should be commented and given Godocs, even
  private methods and functions. It may *seem* obvious what they do at the
  time and why, but you might forget, and others will certainly come along.

* **Test Behaviors**: Test cases should be comprehensible as sets of
  expected behaviors.  Test cases read without code (e.g. just using `It`,
  `Describe`, `Context`, and `By` lines) should still be able to explain
  what's required of the tested interface. Testing behaviors makes
  internal refactors easier, and makes reading tests easier.

* **Real Components Over Mocks**: Avoid mocks and recording actions. Mocks
  tend to be brittle and gradually become more complicated over time (e.g.
  fake client implementations tend to grow into poorly-written, incomplete
  API servers).  Recording of actions tends to lead to brittle tests that
  requires changes during refactors.  Instead, test that the end desired
  state is correct.  Test the way the world should be, without caring how
  it got there, and provide easy ways to set up the real components so
  that mocks aren't required.
