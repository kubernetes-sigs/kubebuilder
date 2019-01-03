# Running GitBook

1. Follow the instructions at [https://toolchain.gitbook.com/](https://toolchain.gitbook.com/) to
  install gitbook.
1. cd into the `docs/book` directory
1. Run `gitbook install`
1. Run `gitbook build`
1. Run `gitbook serve`
1. Visit `http://localhost:4000`

# Steps to deploy

- cd into 'docs/book' directory
- Copy content from '_book' directory into 'public' directory
- Run `firebase deploy --project kubebuilder`
