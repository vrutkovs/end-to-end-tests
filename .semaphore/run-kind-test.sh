#!/bin/bash

set -e

# Load environment variables for VictoriaMetrics components
# These should be set in Semaphore dashboard or secret
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

if [ -n "${LICENSE_FILE}" ]; then
  EXTRA_FLAGS="${EXTRA_FLAGS} --license-file=${LICENSE_FILE}"
fi

# Install Ingress NGINX for Kind
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl delete -A ValidatingWebhookConfiguration ingress-nginx-admission || true

# Wait for ingress to be ready
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s || true

# Run tests
REPORT_DIR="/tmp/allure-results/kind-smoke-test"
mkdir -p "${REPORT_DIR}"

export PATH="$PATH:$(go env GOPATH)/bin"
go install github.com/onsi/ginkgo/v2/ginkgo@${GINKGO_VERSION}

ginkgo -v \
  -procs=1 \
  -timeout=60m \
  --label-filter=kind \
  ./tests/smoke_test \
  -- \
  -env-k8s-distro=kind \
  $EXTRA_FLAGS \
  -report="${REPORT_DIR}"
