#!/bin/bash

set -e

TEST_SUITE=$1

if [ -z "$TEST_SUITE" ]; then
    echo "Usage: $0 <test_suite>"
    exit 1
fi

TF_DIR="terraform/gke"
CLUSTER_NAME="${TEST_SUITE}-${SEMAPHORE_WORKFLOW_NUMBER}"
export GOOGLE_APPLICATION_CREDENTIALS="$HOME"/gcloud-service-key.json

# Terraform cleanup
if [ -d "$TF_DIR" ]; then
    echo "Destroying Terraform resources in ${TF_DIR}..."
    cd "${TF_DIR}"
    terraform init
    # We need the variables to destroy correctly if they are used in the provider or resource naming
    terraform destroy -auto-approve -var="cluster_name=${CLUSTER_NAME}"
    cd -
else
    echo "Terraform directory ${TF_DIR} not found, skipping terraform destroy."
fi

# GCP Cleanup: Remove unused disks that might have been left behind by PersistentVolumeClaims
echo "Cleaning up unused disks in ${GCP_REGION}..."

# We attempt to delete disks in zones a, b, and c of the configured region
for zone_suffix in a b c; do
    ZONE="${GCP_REGION}${zone_suffix}"
    echo "Checking zone ${ZONE}..."

    # List disks that are not currently in use by any resource
    UNUSED_DISKS=$(gcloud compute disks list --filter="-users:*" --format "value(name)" --zones="${ZONE}" 2>/dev/null || true)

    if [ -n "$UNUSED_DISKS" ]; then
        echo "Deleting unused disks in ${ZONE}: ${UNUSED_DISKS}"
        # Split by newline/space and delete
        echo "${UNUSED_DISKS}" | xargs -r gcloud compute disks delete --quiet --zone="${ZONE}" || true
    else
        echo "No unused disks found in ${ZONE}."
    fi
done

echo "Cleanup complete."
