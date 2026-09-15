#!/usr/bin/env bash
#
# Point a Railway service at a new image tag and trigger a redeploy.
#
# Required env:
#   RAILWAY_TOKEN   Railway project token (scoped to one project+environment)
#   SERVICE_ID      Railway service ID
#   ENVIRONMENT_ID  Railway environment ID (must match the token's scope)
#   IMAGE           Fully-qualified image reference (e.g. ghcr.io/org/name:1.2.3)
#
# Optional env:
#   DEPLOY_TIMEOUT_SECONDS  How long to wait for the rollout (default 600)
#   POLL_INTERVAL_SECONDS   Seconds between status checks (default 10)

set -euo pipefail

# Railway deployment statuses, split into terminal-good, terminal-bad, and
# everything else (still rolling out).
classify_status() {
  case "$1" in
    SUCCESS) echo done ;;
    FAILED | CRASHED | REMOVED | SKIPPED) echo bad ;;
    *) echo wait ;;
  esac
}

# Offline check of the branch above: bash railway-deploy.sh --self-test
if [ "${1:-}" = "--self-test" ]; then
  for case_ in "SUCCESS done" "CRASHED bad" "FAILED bad" "REMOVED bad" \
    "SKIPPED bad" "BUILDING wait" "DEPLOYING wait" "INITIALIZING wait"; do
    set -- $case_
    got=$(classify_status "$1")
    [ "$got" = "$2" ] || { echo "classify_status $1 = $got, want $2" >&2; exit 1; }
  done
  echo "self-test ok"
  exit 0
fi

: "${RAILWAY_TOKEN:?RAILWAY_TOKEN is required}"
: "${SERVICE_ID:?SERVICE_ID is required}"
: "${ENVIRONMENT_ID:?ENVIRONMENT_ID is required}"
: "${IMAGE:?IMAGE is required}"

DEPLOY_TIMEOUT_SECONDS="${DEPLOY_TIMEOUT_SECONDS:-600}"
POLL_INTERVAL_SECONDS="${POLL_INTERVAL_SECONDS:-10}"

API_URL="https://backboard.railway.com/graphql/v2"

# Project tokens use the Project-Access-Token header, not Authorization: Bearer.
call_graphql() {
  local payload="$1"
  local label="$2"
  local response
  response=$(curl -sS -X POST "$API_URL" \
    -H "Project-Access-Token: $RAILWAY_TOKEN" \
    -H "Content-Type: application/json" \
    -d "$payload")
  echo "$label response: $response" >&2
  if echo "$response" | jq -e '.errors' >/dev/null 2>&1; then
    echo "$label failed" >&2
    return 1
  fi
  echo "$response"
}

update_payload=$(jq -n \
  --arg serviceId "$SERVICE_ID" \
  --arg environmentId "$ENVIRONMENT_ID" \
  --arg image "$IMAGE" \
  '{
    query: "mutation($serviceId: String!, $environmentId: String!, $input: ServiceInstanceUpdateInput!) { serviceInstanceUpdate(serviceId: $serviceId, environmentId: $environmentId, input: $input) }",
    variables: {
      serviceId: $serviceId,
      environmentId: $environmentId,
      input: { source: { image: $image } }
    }
  }')

echo "Updating Railway service $SERVICE_ID to image $IMAGE"
call_graphql "$update_payload" "serviceInstanceUpdate"

deploy_payload=$(jq -n \
  --arg serviceId "$SERVICE_ID" \
  --arg environmentId "$ENVIRONMENT_ID" \
  '{
    query: "mutation($serviceId: String!, $environmentId: String!) { serviceInstanceDeployV2(serviceId: $serviceId, environmentId: $environmentId) }",
    variables: { serviceId: $serviceId, environmentId: $environmentId }
  }')

echo "Triggering Railway redeploy for service $SERVICE_ID"
deploy_response=$(call_graphql "$deploy_payload" "serviceInstanceDeployV2")
deployment_id=$(echo "$deploy_response" | jq -r '.data.serviceInstanceDeployV2')

if [ -z "$deployment_id" ] || [ "$deployment_id" = "null" ]; then
  echo "Railway did not return a deployment ID" >&2
  exit 1
fi

# Requesting a deploy only queues it. A container that crash-loops on boot
# leaves the previous one serving, so without this wait a broken release
# reports success and the old image keeps running unnoticed.
status_payload=$(jq -n \
  --arg id "$deployment_id" \
  '{
    query: "query($id: String!) { deployment(id: $id) { status } }",
    variables: { id: $id }
  }')

echo "Waiting for deployment $deployment_id (timeout ${DEPLOY_TIMEOUT_SECONDS}s)"
deadline=$((SECONDS + DEPLOY_TIMEOUT_SECONDS))

while true; do
  status=$(call_graphql "$status_payload" "deployment" | jq -r '.data.deployment.status')
  case "$(classify_status "$status")" in
    done)
      echo "Deployment $deployment_id succeeded ($status)."
      exit 0
      ;;
    bad)
      echo "Deployment $deployment_id ended in status $status" >&2
      echo "Check container logs: railway logs $deployment_id --service <name> -d" >&2
      exit 1
      ;;
  esac

  if [ "$SECONDS" -ge "$deadline" ]; then
    echo "Timed out after ${DEPLOY_TIMEOUT_SECONDS}s; last status: $status" >&2
    exit 1
  fi

  sleep "$POLL_INTERVAL_SECONDS"
done
