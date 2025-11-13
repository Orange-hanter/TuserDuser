#!/usr/bin/env bash
set -euo pipefail

# Simple scenario to test email sending via registration endpoint
# Requires the server running locally with EMAIL_PROVIDER=mock (default)
# Usage: ./scripts/test-email.sh [email] [verification_type] [host]
#   email: target email address (default: test+$(date +%s)@example.com)
#   verification_type: email|sms|both (default: email)
#   host: host:port (default: localhost:8080)

EMAIL="${1:-test+$(date +%s)@example.com}"
VERIFICATION_TYPE="${2:-email}"
HOST="${3:-localhost:8080}"

PHONE="+79990000000"
PASSWORD="Test$(date +%s)!aA1"

echo "➡️  Hitting register endpoint to trigger email (provider=mock)"
echo "    Host:       $HOST"
echo "    Email:      $EMAIL"
echo "    Phone:      $PHONE"
echo "    VerifyType: $VERIFICATION_TYPE"

set -x
curl -sS -X POST "http://$HOST/v1/api/auth/register" \
	-H "Content-Type: application/json" \
	-d "{\"email\":\"$EMAIL\",\"phone\":\"$PHONE\",\"password\":\"$PASSWORD\",\"verification_type\":\"$VERIFICATION_TYPE\"}" | jq .
set +x

echo ""
echo "✅ If EMAIL_PROVIDER=mock, check server stdout for lines containing:"
echo "   \"[MOCK] HTML Email отправлен\" and the recipient $EMAIL"
echo "   You should also see the verification code in the HTTP response under 'verify_code'."
echo ""
echo "Tips:"
echo "- Run server locally: 'make run' (ensure EMAIL_PROVIDER=mock in .env)"
echo "- Re-run this script to send another test email"
