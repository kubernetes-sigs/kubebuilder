# Running GitBook

1. Follow the instructions at [https://toolchain.gitbook.com/](https://toolchain.gitbook.com/) to
  install gitbook.
1. cd into the `docs/book` directory
1. Run `gitbook install`
1. Run `gitbook build`
1. Run `gitbook serve`
1. Visit `http://localhost:4000`

# Steps to deploy

There are no manual steps needed to deploy the website.

Kubebuilder book website is deployed on Netlify.
There is a preview of the website for each PR.
As soon as the PR is merged, the website will be built and deployed on Netlify.
