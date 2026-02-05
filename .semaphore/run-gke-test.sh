#!/bin/bash

set -e

TEST_SUITE=$1
PROCS=$2
TIMEOUT=$3

# Load environment variables for VictoriaMetrics components
# These should be set in Semaphore dashboard or environment
EXTRA_FLAGS="-operator-registry=${OPERATOR_REGISTRY} \
  -operator-repository=${OPERATOR_REPOSITORY} \
  -operator-tag=${OPERATOR_TAG} \
  -vm-vmsingledefault-image=${VM_VMSINGLEDEFAULT_IMAGE} \
  -vm-vmsingledefault-version=${VM_VMSINGLEDEFAULT_VERSION} \
  -vm-vmclusterdefault-vmselectdefault-image=${VM_VMCLUSTERDEFAULT_VMSELECTDEFAULT_IMAGE} \
  -vm-vmclusterdefault-vmselectdefault-version=${VM_VMCLUSTERDEFAULT_VMSELECTDEFAULT_VERSION} \
  -vm-vmclusterdefault-vmstoragedefault-image=${VM_VMCLUSTERDEFAULT_VMSTORAGEDEFAULT_IMAGE} \
  -vm-vmclusterdefault-vmstoragedefault-version=${VM_VMCLUSTERDEFAULT_VMSTORAGEDEFAULT_VERSION} \
  -vm-vmclusterdefault-vminsertdefault-image=${VM_VMCLUSTERDEFAULT_VMINSERTDEFAULT_IMAGE} \
  -vm-vmclusterdefault-vminsertdefault-version=${VM_VMCLUSTERDEFAULT_VMINSERTDEFAULT_VERSION} \
  -vm-vmagentdefault-image=${VM_VMAGENTDEFAULT_IMAGE} \
  -vm-vmagentdefault-version=${VM_VMAGENTDEFAULT_VERSION} \
  -vm-vmalertdefault-image=${VM_VMALERTDEFAULT_IMAGE} \
  -vm-vmalertdefault-version=${VM_VMALERTDEFAULT_VERSION} \
  -vm-vmauthdefault-image=${VM_VMAUTHDEFAULT_IMAGE} \
  -vm-vmauthdefault-version=${VM_VMAUTHDEFAULT_VERSION}"

# Install dependencies
echo "Installing dependencies..."
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

wget https://github.com/crust-gather/crust-gather/releases/download/v0.11.1/kubectl-crust-gather_0.11.1_linux_amd64.tar.gz
tar -xvf kubectl-crust-gather_0.11.1_linux_amd64.tar.gz
sudo mv kubectl-crust-gather /usr/local/bin/

wget https://github.com/VictoriaMetrics/vmgather/releases/download/v1.5.0/vmgather-v1.5.0-linux-amd64
sudo mv vmgather-v1.5.0-linux-amd64 /usr/local/bin/vmexporter
sudo chmod +x /usr/local/bin/vmexporter

# GKE Setup
echo "Setting up GKE..."
curl -LO "https://dl.k8s.io/release/v1.35.0/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
gcloud auth activate-service-account --key-file="$HOME"/gcloud-service-key.json
export GOOGLE_APPLICATION_CREDENTIALS="$HOME"/gcloud-service-key.json

# Terraform Setup
echo "Provisioning infrastructure with Terraform..."
echo "${TF_VAR_BASE64}" | base64 --decode > terraform/gke/terraform.tfvars

CLUSTER_NAME="${TEST_SUITE}-${SEMAPHORE_WORKFLOW_NUMBER}"

cd terraform/gke
terraform init
terraform apply -auto-approve -var="cluster_name=${CLUSTER_NAME}"
cd -

# Update Kubeconfig
echo "Updating kubeconfig..."
gcloud container clusters get-credentials "${CLUSTER_NAME}" --region=europe-central2 --project="${PROJECT_ID}"

# Create admin service account for testing
kubectl -n kube-system create serviceaccount cluster-admin || true
kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --serviceaccount=kube-system:cluster-admin || true
kubectl -n kube-system create token --duration=24h cluster-admin > /tmp/token.txt
kubectl config view --raw --minify -o jsonpath='{.clusters[0].cluster.certificate-authority-data}' | base64 -d > /tmp/ca.txt
kubectl config view --raw --minify -o jsonpath='{.clusters[0].cluster.server}' > /tmp/server.txt

export KUBECONFIG=/tmp/kubeconfig.yaml
kubectl config set-cluster gke --server=$(cat /tmp/server.txt) --certificate-authority=/tmp/ca.txt --embed-certs=true
kubectl config set-credentials cluster-admin --token=$(cat /tmp/token.txt)
kubectl config set-context production --cluster gke --user cluster-admin
kubectl config use-context production

# Install Ingress
echo "Installing Ingress..."
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl delete -A ValidatingWebhookConfiguration ingress-nginx-admission || true

# Add Helm repos
echo "Adding Helm repositories..."
helm repo add vm https://victoriametrics.github.io/helm-charts/
helm repo add chaos-mesh https://charts.chaos-mesh.org
helm repo update

# Run tests
echo "Starting Ginkgo tests for ${TEST_SUITE}..."
REPORT_DIR="/tmp/allure-results/${TEST_SUITE}"
mkdir -p "${REPORT_DIR}"

export PATH="$PATH:$(go env GOPATH)/bin"
go install github.com/onsi/ginkgo/v2/ginkgo@v2.27.4

ginkgo -v \
  -procs="${PROCS}" \
  -timeout="${TIMEOUT}" \
  --label-filter=gke \
  "./tests/${TEST_SUITE}_test" \
  -- \
  -env-k8s-distro=gke \
  $EXTRA_FLAGS \
  -report="${REPORT_DIR}"
