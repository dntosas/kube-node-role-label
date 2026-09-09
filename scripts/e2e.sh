#!/usr/bin/env bash
# End-to-end test against a kind cluster.
#
# Builds the image, loads it into kind, installs the chart and asserts that a
# worker node labelled node-type=worker ends up with the "worker" role while
# the control-plane node is left alone.
#
# Requires: docker, kind, kubectl, helm, go. Creates the cluster unless
# KIND_CLUSTER points to an existing one.
set -euo pipefail

KIND_CLUSTER="${KIND_CLUSTER:-knrl-e2e}"
IMAGE="${IMAGE:-ghcr.io/dntosas/kube-node-role-label:e2e}"
NAMESPACE="${NAMESPACE:-kube-utils}"
RELEASE="kube-node-role-label"
CHART="$(cd "$(dirname "$0")/.." && pwd)/charts/kube-node-role-label"
KEEP_CLUSTER="${KEEP_CLUSTER:-false}"

log() { printf '\n==> %s\n' "$*"; }

cleanup() {
  if [[ "${KEEP_CLUSTER}" != "true" && "${CREATED_CLUSTER:-false}" == "true" ]]; then
    log "Deleting kind cluster ${KIND_CLUSTER}"
    kind delete cluster --name "${KIND_CLUSTER}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

if ! kind get clusters 2>/dev/null | grep -qx "${KIND_CLUSTER}"; then
  log "Creating kind cluster ${KIND_CLUSTER}"
  kind create cluster --name "${KIND_CLUSTER}" --wait 120s \
    --config "$(dirname "$0")/../.github/config/kind.yaml"
  CREATED_CLUSTER=true
fi
kubectl config use-context "kind-${KIND_CLUSTER}" >/dev/null

log "Building image ${IMAGE}"
"$(dirname "$0")/docker-build.sh" "${IMAGE}"
kind load docker-image --name "${KIND_CLUSTER}" "${IMAGE}"

WORKER="$(kubectl get nodes -l '!node-role.kubernetes.io/control-plane' -o jsonpath='{.items[0].metadata.name}')"
OTHER_WORKER="$(kubectl get nodes -l '!node-role.kubernetes.io/control-plane' -o jsonpath='{.items[1].metadata.name}')"
CONTROL_PLANE="$(kubectl get nodes -l 'node-role.kubernetes.io/control-plane' -o jsonpath='{.items[0].metadata.name}')"

log "Labelling ${WORKER} node-type=worker and ${CONTROL_PLANE} node-type=cp"
kubectl label node "${WORKER}" node-type=worker --overwrite
kubectl label node "${CONTROL_PLANE}" node-type=cp --overwrite

log "Installing chart"
helm upgrade --install "${RELEASE}" "${CHART}" \
  --namespace "${NAMESPACE}" --create-namespace \
  --set image.repository="${IMAGE%:*}" \
  --set image.tag="${IMAGE##*:}" \
  --set image.pullPolicy=IfNotPresent \
  --set label_watch.interval=5s \
  --set label_watch.labels=node-type \
  --set logLevel=debug \
  --wait --timeout 120s

log "Waiting for ${WORKER} to receive node-role.kubernetes.io/worker=true"
for _ in $(seq 1 30); do
  if [[ "$(kubectl get node "${WORKER}" -o jsonpath='{.metadata.labels.node-role\.kubernetes\.io/worker}')" == "true" ]]; then
    break
  fi
  sleep 2
done

kubectl -n "${NAMESPACE}" logs deploy/"${RELEASE}" --tail=50

fail=0
check() {
  local desc="$1" got="$2" want="$3"
  if [[ "${got}" == "${want}" ]]; then
    echo "PASS ${desc}"
  else
    echo "FAIL ${desc}: got '${got}', want '${want}'"
    fail=1
  fi
}

check "worker gets role" \
  "$(kubectl get node "${WORKER}" -o jsonpath='{.metadata.labels.node-role\.kubernetes\.io/worker}')" "true"
check "unlabelled worker untouched" \
  "$(kubectl get node "${OTHER_WORKER}" -o jsonpath='{.metadata.labels.node-role\.kubernetes\.io/worker}')" ""
check "control-plane untouched" \
  "$(kubectl get node "${CONTROL_PLANE}" -o jsonpath='{.metadata.labels.node-role\.kubernetes\.io/cp}')" ""

log "Checking steady state is quiet (no info-level 'node role set' after the first pass)"
sleep 12
repeats="$(kubectl -n "${NAMESPACE}" logs deploy/"${RELEASE}" --since=10s | grep -c '"msg":"node role set"' || true)"
check "no repeated success logs" "${repeats}" "0"

kubectl get nodes
exit "${fail}"
