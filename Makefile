# Makefile for VictoriaMetrics End-to-End Tests

# Dependencies versions
GO_VERSION ?= 1.25.3
KIND_VERSION ?= v0.31.0
KUBECTL_VERSION ?= v1.35.0
CRUST_GATHER_VERSION ?= v0.11.1
VMGATHER_VERSION ?= v1.5.0
GINKGO_VERSION ?= v2.27.4

# Image versions
OPERATOR_REGISTRY ?= quay.io
OPERATOR_REPOSITORY ?= victoriametrics/operator
OPERATOR_TAG ?= v0.67.0

VM_VMSINGLEDEFAULT_VERSION ?= v1.122.14-enterprise
VM_VMCLUSTERDEFAULT_VERSION ?= v1.122.14-cluster-enterprise

VM_VMSINGLEDEFAULT_IMAGE ?= quay.io/victoriametrics/victoria-metrics

VM_VMCLUSTERDEFAULT_VMSELECTDEFAULT_IMAGE ?= quay.io/victoriametrics/vmselect
VM_VMCLUSTERDEFAULT_VMSELECTDEFAULT_VERSION ?= $(VM_VMCLUSTERDEFAULT_VERSION)

VM_VMCLUSTERDEFAULT_VMSTORAGEDEFAULT_IMAGE ?= quay.io/victoriametrics/vmstorage
VM_VMCLUSTERDEFAULT_VMSTORAGEDEFAULT_VERSION ?= $(VM_VMCLUSTERDEFAULT_VERSION)

VM_VMCLUSTERDEFAULT_VMINSERTDEFAULT_IMAGE ?= quay.io/victoriametrics/vminsert
VM_VMCLUSTERDEFAULT_VMINSERTDEFAULT_VERSION ?= $(VM_VMCLUSTERDEFAULT_VERSION)

VM_VMAGENTDEFAULT_IMAGE ?= quay.io/victoriametrics/vmagent
VM_VMAGENTDEFAULT_VERSION ?= $(VM_VMSINGLEDEFAULT_VERSION)

VM_VMALERTDEFAULT_IMAGE ?= quay.io/victoriametrics/vmalert
VM_VMALERTDEFAULT_VERSION ?= $(VM_VMSINGLEDEFAULT_VERSION)

VM_VMAUTHDEFAULT_IMAGE ?= quay.io/victoriametrics/vmauth
VM_VMAUTHDEFAULT_VERSION ?= $(VM_VMSINGLEDEFAULT_VERSION)

VM_VMBACKUPDEFAULT_IMAGE ?= quay.io/victoriametrics/vmbackup
VM_VMBACKUPDEFAULT_VERSION ?= $(VM_VMSINGLEDEFAULT_VERSION)

VM_VMRESTOREDEFAULT_IMAGE ?= quay.io/victoriametrics/vmrestore
VM_VMRESTOREDEFAULT_VERSION ?= $(VM_VMSINGLEDEFAULT_VERSION)

LICENSE_FILE ?=

# Configuration
BIN_DIR := $(shell pwd)/bin
GOPATH_BIN := $(shell go env GOPATH)/bin
export PATH := $(BIN_DIR):$(GOPATH_BIN):$(PATH)
GCP_REGION ?= europe-central2

# Test configuration
TEST_SUITE ?= smoke
PROCS ?= 1
TIMEOUT ?= 60m
REPORT_DIR ?= /tmp/allure-results
SEMAPHORE_WORKFLOW_NUMBER ?= 0

EXTRA_FLAGS := -operator-registry=$(OPERATOR_REGISTRY) \
	-operator-repository=$(OPERATOR_REPOSITORY) \
	-operator-tag=$(OPERATOR_TAG) \
	-vm-vmsingledefault-image=$(VM_VMSINGLEDEFAULT_IMAGE) \
	-vm-vmsingledefault-version=$(VM_VMSINGLEDEFAULT_VERSION) \
	-vm-vmclusterdefault-vmselectdefault-image=$(VM_VMCLUSTERDEFAULT_VMSELECTDEFAULT_IMAGE) \
	-vm-vmclusterdefault-vmselectdefault-version=$(VM_VMCLUSTERDEFAULT_VMSELECTDEFAULT_VERSION) \
	-vm-vmclusterdefault-vmstoragedefault-image=$(VM_VMCLUSTERDEFAULT_VMSTORAGEDEFAULT_IMAGE) \
	-vm-vmclusterdefault-vmstoragedefault-version=$(VM_VMCLUSTERDEFAULT_VMSTORAGEDEFAULT_VERSION) \
	-vm-vmclusterdefault-vminsertdefault-image=$(VM_VMCLUSTERDEFAULT_VMINSERTDEFAULT_IMAGE) \
	-vm-vmclusterdefault-vminsertdefault-version=$(VM_VMCLUSTERDEFAULT_VMINSERTDEFAULT_VERSION) \
	-vm-vmagentdefault-image=$(VM_VMAGENTDEFAULT_IMAGE) \
	-vm-vmagentdefault-version=$(VM_VMAGENTDEFAULT_VERSION) \
	-vm-vmalertdefault-image=$(VM_VMALERTDEFAULT_IMAGE) \
	-vm-vmalertdefault-version=$(VM_VMALERTDEFAULT_VERSION) \
	-vm-vmauthdefault-image=$(VM_VMAUTHDEFAULT_IMAGE) \
	-vm-vmauthdefault-version=$(VM_VMAUTHDEFAULT_VERSION)

ifneq ($(LICENSE_FILE),)
	EXTRA_FLAGS += --license-file=$(LICENSE_FILE)
endif

# Targets
.PHONY: all
all: install-dependencies

.PHONY: install-dependencies
install-dependencies: install-kubectl install-helm install-kind install-crust-gather install-vmexporter install-ginkgo

.PHONY: install-kubectl
install-kubectl:
	@mkdir -p $(BIN_DIR)
	curl -LO "https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/linux/amd64/kubectl"
	chmod +x kubectl
	mv kubectl $(BIN_DIR)/

.PHONY: install-helm
install-helm:
	@mkdir -p $(BIN_DIR)
	curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | HELM_INSTALL_DIR=$(BIN_DIR) bash -s -- --no-sudo
	$(BIN_DIR)/helm repo add vm https://victoriametrics.github.io/helm-charts/
	$(BIN_DIR)/helm repo add chaos-mesh https://charts.chaos-mesh.org
	$(BIN_DIR)/helm repo update

.PHONY: install-kind
install-kind:
	@mkdir -p $(BIN_DIR)
	curl -Lo ./kind https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-linux-amd64
	chmod +x ./kind
	mv ./kind $(BIN_DIR)/

.PHONY: install-crust-gather
install-crust-gather:
	@mkdir -p $(BIN_DIR)
	wget https://github.com/crust-gather/crust-gather/releases/download/$(CRUST_GATHER_VERSION)/kubectl-crust-gather_$(patsubst v%,%,$(CRUST_GATHER_VERSION))_linux_amd64.tar.gz
	tar -xvf kubectl-crust-gather_$(patsubst v%,%,$(CRUST_GATHER_VERSION))_linux_amd64.tar.gz
	mv kubectl-crust-gather $(BIN_DIR)/
	rm kubectl-crust-gather_$(patsubst v%,%,$(CRUST_GATHER_VERSION))_linux_amd64.tar.gz

.PHONY: install-vmexporter
install-vmexporter:
	@mkdir -p $(BIN_DIR)
	wget https://github.com/VictoriaMetrics/vmgather/releases/download/$(VMGATHER_VERSION)/vmgather-$(VMGATHER_VERSION)-linux-amd64
	mv vmgather-$(VMGATHER_VERSION)-linux-amd64 $(BIN_DIR)/vmexporter
	chmod +x $(BIN_DIR)/vmexporter

.PHONY: install-ginkgo
install-ginkgo:
	go install github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION)

.PHONY: install-ingress
install-ingress: install-kubectl
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
	kubectl delete -A ValidatingWebhookConfiguration ingress-nginx-admission || true
	# Wait for ingress to be ready
	kubectl wait --namespace ingress-nginx \
	  --for=condition=ready pod \
	  --selector=app.kubernetes.io/component=controller \
	  --timeout=90s || true

# Unit tests
.PHONY: test-unit
test-unit:
	go mod download
	go test ./pkg/... -v -failfast

# Kind targets
.PHONY: kind-create
kind-create: install-kind
	kind get clusters | grep -q kind || kind create cluster --config manifests/kind.yaml

.PHONY: kind-delete
kind-delete:
	kind delete cluster

.PHONY: test-kind
test-kind: install-dependencies kind-create
	$(MAKE) install-ingress
	@mkdir -p $(REPORT_DIR)/kind-smoke-test
	ginkgo -v \
		-procs=1 \
		-timeout=60m \
		--label-filter=kind \
		./tests/smoke_test \
		-- \
		-env-k8s-distro=kind \
		$(EXTRA_FLAGS) \
		-report="$(REPORT_DIR)/kind-smoke-test"

# GKE targets
.PHONY: test-gke
test-gke: install-dependencies
	$(MAKE) gke-provision
	$(MAKE) gke-prepare-access
	$(MAKE) install-ingress
	$(MAKE) gke-run-test

.PHONY: gcloud-auth
gcloud-auth:
	@if [ -f "$(HOME)/gcloud-service-key.json" ]; then \
		gcloud auth activate-service-account --key-file="$(HOME)/gcloud-service-key.json"; \
	fi

.PHONY: gke-provision
gke-provision: gcloud-auth
	@if [ -z "$(TF_VAR_BASE64)" ]; then echo "TF_VAR_BASE64 is not set"; exit 1; fi
	echo "$(TF_VAR_BASE64)" | base64 --decode > terraform/gke/terraform.tfvars
	cd terraform/gke && \
		export GOOGLE_APPLICATION_CREDENTIALS="$(HOME)/gcloud-service-key.json" && \
		terraform init && \
		terraform apply -auto-approve -var="cluster_name=$(TEST_SUITE)-$(SEMAPHORE_WORKFLOW_NUMBER)"

.PHONY: gke-prepare-access
gke-prepare-access: gcloud-auth
	@if [ -z "$(PROJECT_ID)" ]; then echo "PROJECT_ID is not set"; exit 1; fi
	gcloud container clusters get-credentials "$(TEST_SUITE)-$(SEMAPHORE_WORKFLOW_NUMBER)" --region=$(GCP_REGION) --project="$(PROJECT_ID)"
	kubectl -n kube-system create serviceaccount cluster-admin || true
	kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --serviceaccount=kube-system:cluster-admin || true
	# Generate dedicated kubeconfig for test
	kubectl -n kube-system create token --duration=24h cluster-admin > /tmp/token.txt
	kubectl config view --raw --minify -o jsonpath='{.clusters[0].cluster.certificate-authority-data}' | base64 -d > /tmp/ca.txt
	kubectl config view --raw --minify -o jsonpath='{.clusters[0].cluster.server}' > /tmp/server.txt

.PHONY: gke-run-test
gke-run-test:
	@mkdir -p $(REPORT_DIR)/$(TEST_SUITE)
	# Setup isolated kubeconfig if files exist
	if [ -f /tmp/token.txt ]; then \
		export KUBECONFIG=/tmp/kubeconfig.yaml; \
		kubectl config set-cluster gke --server=$$(cat /tmp/server.txt) --certificate-authority=/tmp/ca.txt --embed-certs=true; \
		kubectl config set-credentials cluster-admin --token=$$(cat /tmp/token.txt); \
		kubectl config set-context production --cluster gke --user cluster-admin; \
		kubectl config use-context production; \
	fi; \
	ginkgo -v \
		-procs=$(PROCS) \
		-timeout=$(TIMEOUT) \
		--label-filter=gke \
		"./tests/$(TEST_SUITE)_test" \
		-- \
		-env-k8s-distro=gke \
		$(EXTRA_FLAGS) \
		-report="$(REPORT_DIR)/$(TEST_SUITE)"

.PHONY: clean-gke
clean-gke: gcloud-auth
	cd terraform/gke && \
		export GOOGLE_APPLICATION_CREDENTIALS="$(HOME)/gcloud-service-key.json" && \
		terraform init && \
		terraform destroy -auto-approve -var="cluster_name=$(TEST_SUITE)-$(SEMAPHORE_WORKFLOW_NUMBER)"
	# Disk cleanup
	@echo "Cleaning up unused disks in $(GCP_REGION)..."
	@for zone_suffix in a b c; do \
		ZONE="$(GCP_REGION)$$zone_suffix"; \
		echo "Checking zone $$ZONE..."; \
		UNUSED_DISKS=$$(gcloud compute disks list --filter="-users:*" --format "value(name)" --zones="$$ZONE" 2>/dev/null || true); \
		if [ -n "$$UNUSED_DISKS" ]; then \
			echo "Deleting unused disks in $$ZONE: $$UNUSED_DISKS"; \
			echo "$$UNUSED_DISKS" | xargs -r gcloud compute disks delete --quiet --zone="$$ZONE" || true; \
		else \
			echo "No unused disks found in $$ZONE."; \
		fi; \
	done
