#!/usr/bin/env bash
#
# Point a Railway service at a new image tag and trigger a redeploy.
#
# Required env:
#   RAILWAY_TOKEN   Railway project token (scoped to one project+environment)
#   SERVICE_ID      Railway service ID
#   ENVIRONMENT_ID  Railway environment ID (must match the token's scope)
#   IMAGE           Fully-qualified image reference (e.g. ghcr.io/org/name:1.2.3)

set -euo pipefail

: "${RAILWAY_TOKEN:?RAILWAY_TOKEN is required}"
: "${SERVICE_ID:?SERVICE_ID is required}"
: "${ENVIRONMENT_ID:?ENVIRONMENT_ID is required}"
: "${IMAGE:?IMAGE is required}"

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
  echo "$label response: $response"
  if echo "$response" | jq -e '.errors' >/dev/null; then
    echo "$label failed" >&2
    exit 1
  fi
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
call_graphql "$deploy_payload" "serviceInstanceDeployV2"

echo "Railway deploy requested successfully."
