#!/usr/bin/env zsh
# Log in as admin and grant a user the creator role.
#
# Usage:
#   ADMIN_PASSWORD=secret ./scripts/grant_creator.sh 0a8b570c-2890-23e3-c6b1-460b665a1f03
#   BASE_URL=https://api.tuserduser.online/v1 ADMIN_EMAIL=admin@domain ADMIN_PASSWORD=... ./scripts/grant_creator.sh <user-id> [-r support]

set -euo pipefail

if [ $# -lt 1 ]; then
	echo "Usage: $0 <user-id> [role]"
	exit 1
fi

USER_ID=$1
ROLE=${2:-creator}

BASE_URL=${BASE_URL:-http://localhost:8080/v1}
DOTENV_FILE=${DOTENV_FILE:-$(cd "$(dirname -- "$0")/.." && pwd)/.env}

load_env_value() {
	local key=$1
	local value
	value=$(grep -E "^${key}=" "$DOTENV_FILE" 2>/dev/null | tail -n1 | cut -d'=' -f2-)
	value=${value%%$'\r'}
	printf '%s' "$value"
}

if [ -f "$DOTENV_FILE" ]; then
	ADMIN_EMAIL=${ADMIN_EMAIL:-$(load_env_value ADMIN_EMAIL)}
	ADMIN_PASSWORD=${ADMIN_PASSWORD:-$(load_env_value ADMIN_PASSWORD)}
fi

ADMIN_EMAIL=${ADMIN_EMAIL:-admin@example.com}

if [ -z "${ADMIN_PASSWORD-}" ]; then
	echo "ERROR: ADMIN_PASSWORD environment variable is required to log in as admin"
	exit 2
fi

CONTENT_TYPE="application/json"

login_payload=$(
	cat <<EOF
{
  "email": "${ADMIN_EMAIL}",
  "password": "${ADMIN_PASSWORD}"
}
EOF
)

echo "[*] Logging in as admin (${ADMIN_EMAIL}) against ${BASE_URL}/api/auth/login"

response=$(curl -sSf -X POST "${BASE_URL}/api/auth/login" \
	-H "Content-Type: ${CONTENT_TYPE}" \
	-d "${login_payload}")

token=$(python3 -c 'import sys, json; data=json.load(sys.stdin); print(data.get("access_token", ""))' <<<"${response}")

if [ -z "$token" ] || [ "$token" = "null" ]; then
	echo "ERROR: failed to obtain access token from login response"
	echo "Response was: $response"
	exit 3
fi

echo "[*] Admin token obtained. Granting '${ROLE}' role to user ${USER_ID}"

grant_payload=$(
	cat <<EOF
{
  "user_id": "${USER_ID}",
  "role": "${ROLE}"
}
EOF
)

grant_response=$(curl -sSf -X PUT "${BASE_URL}/api/admin/users/role" \
	-H "Content-Type: ${CONTENT_TYPE}" \
	-H "Authorization: Bearer ${token}" \
	-d "${grant_payload}")

echo "[+] Role updated. Server responded with:"
echo "$grant_response"
