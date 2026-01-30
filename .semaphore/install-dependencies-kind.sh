#!/bin/bash

set -e

# Install kubectl
curl -LO "https://dl.k8s.io/release/v1.35.0/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/kubectl

# Install helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Install Kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.31.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Add helm repo
helm repo add vm https://victoriametrics.github.io/helm-charts/
helm repo update

# Install crust-gather
wget https://github.com/crust-gather/crust-gather/releases/download/v0.11.1/kubectl-crust-gather_0.11.1_linux_amd64.tar.gz
tar -xvf kubectl-crust-gather_0.11.1_linux_amd64.tar.gz
sudo mv kubectl-crust-gather /usr/local/bin/

# Install VMExporter
wget https://github.com/VictoriaMetrics/vmgather/releases/download/v1.5.0/vmgather-v1.5.0-linux-amd64
sudo mv vmgather-v1.5.0-linux-amd64 /usr/local/bin/vmexporter
sudo chmod +x /usr/local/bin/vmexporter

# Install Ginkgo
export PATH="$PATH:$(go env GOPATH)/bin"
go install github.com/onsi/ginkgo/v2/ginkgo@latest
