#!/usr/bin/env bash
set -eu

# Use DEBUG=1 ./scripts/download-binaries.sh to get debug output
quiet="-s"
[[ -z "${DEBUG:-""}" ]] || {
  set -x
  quiet=""
}

logEnd() {
  local msg='done.'
  [ "$1" -eq 0 ] || msg='Error downloading assets'
  echo "$msg"
}
trap 'logEnd $?' EXIT

# Use BASE_URL=https://my/binaries/url ./scripts/download-binaries to download
# from a different bucket
: "${BASE_URL:="https://storage.googleapis.com/k8s-c10s-test-binaries"}"

test_framework_dir="$(cd "$(dirname "$0")/.." ; pwd)"
os="$(uname -s)"
os_lowercase="$(echo "$os" | tr '[:upper:]' '[:lower:]' )"
arch="$(uname -m)"

dest_dir="${1:-"${test_framework_dir}/assets/bin"}"
etcd_dest="${dest_dir}/etcd"
kubectl_dest="${dest_dir}/kubectl"
kube_apiserver_dest="${dest_dir}/kube-apiserver"

echo "About to download a couple of binaries. This might take a while..."

curl $quiet "${BASE_URL}/etcd-${os}-${arch}" --output "$etcd_dest"
curl $quiet "${BASE_URL}/kube-apiserver-${os}-${arch}" --output "$kube_apiserver_dest"

kubectl_version="$(curl $quiet https://storage.googleapis.com/kubernetes-release/release/stable.txt)"
kubectl_url="https://storage.googleapis.com/kubernetes-release/release/${kubectl_version}/bin/${os_lowercase}/amd64/kubectl"
curl $quiet "$kubectl_url" --output "$kubectl_dest"

chmod +x "$etcd_dest" "$kubectl_dest" "$kube_apiserver_dest"

echo    "# destination:"
echo    "#   ${dest_dir}"
echo    "# versions:"
echo -n "#   etcd:            "; "$etcd_dest" --version | head -n 1
echo -n "#   kube-apiserver:  "; "$kube_apiserver_dest" --version
echo -n "#   kubectl:         "; "$kubectl_dest" version --client --short
