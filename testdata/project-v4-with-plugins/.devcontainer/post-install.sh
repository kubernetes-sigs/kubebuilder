#!/bin/bash
set -x

BASHRC_FILE="$HOME/.bashrc"

curl -Lo ./kind https://kind.sigs.k8s.io/dl/latest/kind-linux-$(go env GOARCH)
chmod +x ./kind
mv ./kind /usr/local/bin/kind

BEGIN_MARKER="# BEGIN kind autocompletion"
END_MARKER="# END kind autocompletion"
if ! grep -q "$BEGIN_MARKER" "$BASHRC_FILE"; then
    echo ""
    echo "" >> "$BASHRC_FILE"
    echo "$BEGIN_MARKER" >> "$BASHRC_FILE"
    echo "# kind autocompletion" >> "$BASHRC_FILE"
    echo "if [ -f /usr/local/share/bash-completion/bash_completion ]; then" >> "$BASHRC_FILE"
    echo ". /usr/local/share/bash-completion/bash_completion" >> "$BASHRC_FILE"
    echo "fi" >> "$BASHRC_FILE"
    echo ". <(kind completion bash)" >> "$BASHRC_FILE"
    echo "$END_MARKER" >> "$BASHRC_FILE"
    echo ""
fi

curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/linux/$(go env GOARCH)
chmod +x kubebuilder
mv kubebuilder /usr/local/bin/

BEGIN_MARKER="# BEGIN kubebuilder autocompletion"
END_MARKER="# END kubebuilder autocompletion"
if ! grep -q "$BEGIN_MARKER" "$BASHRC_FILE"; then
    echo ""
    echo "" >> "$BASHRC_FILE"
    echo "$BEGIN_MARKER" >> "$BASHRC_FILE"
    echo "# kubebuilder autocompletion" >> "$BASHRC_FILE"
    echo "if [ -f /usr/local/share/bash-completion/bash_completion ]; then" >> "$BASHRC_FILE"
    echo ". /usr/local/share/bash-completion/bash_completion" >> "$BASHRC_FILE"
    echo "fi" >> "$BASHRC_FILE"
    echo ". <(kubebuilder completion bash)" >> "$BASHRC_FILE"
    echo "$END_MARKER" >> "$BASHRC_FILE"
    echo ""
fi

KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
curl -LO "https://dl.k8s.io/release/$KUBECTL_VERSION/bin/linux/$(go env GOARCH)/kubectl"
chmod +x kubectl
mv kubectl /usr/local/bin/kubectl

BEGIN_MARKER="# BEGIN kubectl autocompletion"
END_MARKER="# END kubectl autocompletion"
if ! grep -q "$BEGIN_MARKER" "$BASHRC_FILE"; then
    echo ""
    echo "" >> "$BASHRC_FILE"
    echo "$BEGIN_MARKER" >> "$BASHRC_FILE"
    echo "# kubectl autocompletion" >> "$BASHRC_FILE"
    echo "if [ -f /usr/local/share/bash-completion/bash_completion ]; then" >> "$BASHRC_FILE"
    echo ". /usr/local/share/bash-completion/bash_completion" >> "$BASHRC_FILE"
    echo "fi" >> "$BASHRC_FILE"
    echo ". <(kubectl completion bash)" >> "$BASHRC_FILE"
    echo "$END_MARKER" >> "$BASHRC_FILE"
    echo ""
fi

BEGIN_MARKER="# BEGIN docker autocompletion"
END_MARKER="# END docker autocompletion"
if ! grep -q "$BEGIN_MARKER" "$BASHRC_FILE"; then
    echo ""
    echo "" >> "$BASHRC_FILE"
    echo "$BEGIN_MARKER" >> "$BASHRC_FILE"
    echo "# docker autocompletion" >> "$BASHRC_FILE"
    echo "if [ -f /usr/local/share/bash-completion/bash_completion ]; then" >> "$BASHRC_FILE"
    echo ". /usr/local/share/bash-completion/bash_completion" >> "$BASHRC_FILE"
    echo "fi" >> "$BASHRC_FILE"
    echo ". <(docker completion bash)" >> "$BASHRC_FILE"
    echo "$END_MARKER" >> "$BASHRC_FILE"
    echo ""
fi

docker network create -d=bridge --subnet=172.19.0.0/24 kind

kind version
kubebuilder version
docker --version
go version
kubectl version --client
