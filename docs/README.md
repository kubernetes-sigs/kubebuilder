# Running mdBook

The kubebuilder book is served using [mdBook](https://github.com/rust-lang-nursery/mdBook). If you want to test changes to the book locally, follow these directions:

1. Follow the instructions at [https://github.com/rust-lang-nursery/mdBook#installation](https://github.com/rust-lang-nursery/mdBook#installation) to
   install mdBook.
2. Make sure [controller-gen](https://pkg.go.dev/sigs.k8s.io/controller-tools/cmd/controller-gen) is installed in `$GOPATH`.
3. cd into the `docs/book` directory
4. Run `mdbook serve`
5. Visit [http://localhost:3000](http://localhost:3000)

# Steps to deploy

There are no manual steps needed to deploy the website.

Kubebuilder book website is deployed on Netlify.
There is a preview of the website for each PR.
As soon as the PR is merged, the website will be built and deployed on Netlify.
