## Crossplane Resource Pack Builder

This is a fork of Kubebuilder with some modifications to generate & build a Crossplane
Stack for your kustomize or helm charts without needing to write any code.

This is prepared for purely POC purposes. If implemented, the following would
be the only command you need to run after you get your chart ready:

```bash
resourcepacks build . -t myorg/myrepo:1.0.0
```

### Run the Demo

* Download the example kustomize repo. Currently, only kustomize is supported.
```bash
mkdir -p $GOPATH/src/github.com/muvaf
cd $GOPATH/src/github.com/muvaf
git clone https://github.com/muvaf/generated-minimal-gcp.git
```

* Build Kubebuilder and move `bin/kubebuilder` executable into the folder of your
  kustomize repo.
```bash
mkdir -p $GOPATH/src/github.com/muvaf
cd $GOPATH/src/github.com/muvaf
git clone https://github.com/muvaf/kubebuilder.git
cd kubebuilder
make build
chmod +x bin/kubebuilder
cp bin/kubebuilder $GOPATH/src/github.com/muvaf/generated-minimal-gcp/
```

* The following is the set of commands you need to run to generate & build the
  container image of your stack. You can run them one by one or put in a script:
```bash
#!/usr/bin/env bash
set -eE

cd $GOPATH/src/github.com/muvaf/generated-minimal-gcp/

MODULE_NAME=$(basename $(pwd))
COMMIT=$(git rev-parse HEAD)
TMP_DIR="/tmp/stack-${COMMIT}"
mkdir -p "${TMP_DIR}"
trap "rm -rf ${TMP_DIR}" EXIT

rsync -a $GOPATH/src/github.com/muvaf/generated-minimal-gcp/ "${TMP_DIR}" --exclude '.git'
cd ${TMP_DIR}
GO111MODULE=on

go mod init "${MODULE_NAME}"
./kubebuilder init --domain resourcepacks.crossplane.io
./kubebuilder create api \
  --controller=true \
  --example=true \
  --group gcp \
  --version v1alpha1 \
  --kind MinimalGCP \
  --make=true \
  --namespaced=false \
  --resource=true

make manifests

kubectl crossplane stack init --cluster "muvaf/${MODULE_NAME}"
echo "COPY resources/ resources/" >> stack.Dockerfile
kubectl crossplane stack build
```

In the output, you should see a docker image. You can use that image to install
your stack.

### Future

* Kubebuilder is modified for this functionality. Instead, a standalone CLI that
  utilises kubebuilder as a library would be much better.
* Helm support.
* Implementation of features in `Stackfile.yaml.future` file in the example repo,
  most notable one being `Patcher`s.
