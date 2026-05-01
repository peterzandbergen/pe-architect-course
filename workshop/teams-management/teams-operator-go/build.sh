#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE_NAME="${IMAGE_NAME:-teams-operator-go}"
IMAGE_TAG="${IMAGE_TAG:-0.0.1}"
FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"
CLUSTER_NAME="${CLUSTER_NAME:-5min-idp}"
LOAD_KIND="${LOAD_KIND:-true}"

echo "Building ${FULL_IMAGE}..."
docker build -t "${FULL_IMAGE}" "${SCRIPT_DIR}"

if [ "${LOAD_KIND}" = "true" ]; then
  echo "Loading image into kind cluster '${CLUSTER_NAME}'..."
  kind load docker-image "${FULL_IMAGE}" --name "${CLUSTER_NAME}"
else
  echo "Skipping kind load (LOAD_KIND=${LOAD_KIND})."
  echo "To load later: kind load docker-image \"${FULL_IMAGE}\" --name \"${CLUSTER_NAME}\""
fi

echo "Done. Deploy with:"
echo "  kubectl apply -f ${SCRIPT_DIR}/deployment.yaml"
echo ""
echo "Or swap the Python operator by editing operator-deployment.yaml:"
echo "  image: ${FULL_IMAGE}"
echo "  imagePullPolicy: Never"