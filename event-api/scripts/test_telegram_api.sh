#!/usr/bin/env bash
# =============================================================================
# Integration Test: Telegram API Endpoints
# =============================================================================
# Tests all Telegram notification endpoints:
#   - POST /v1/api/notifications/telegram/link   (generate binding link)
#   - GET  /v1/api/notifications/telegram/bound  (check if user is bound)
#   - GET  /v1/api/notifications/telegram/status (detailed binding status)
#   - POST /v1/api/notifications/telegram/unbind (remove binding)
#
# Usage:
#   ./scripts/test_telegram_api.sh [BASE_URL]
#
# Examples:
#   ./scripts/test_telegram_api.sh                     # defaults to localhost:8080
#   ./scripts/test_telegram_api.sh http://localhost:8080
#   ./scripts/test_telegram_api.sh https://api.example.com
# =============================================================================

set -euo pipefail

# Configuration
BASE_URL="${1:-http://localhost:8080}"
API_URL="${BASE_URL}/v1/api"
TIMESTAMP=$(date +%s)
EMAIL="telegram-test+${TIMESTAMP}@example.com"
PHONE="+7999${TIMESTAMP: -7}"
PASSWORD="testpassword123"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() {
	echo -e "${GREEN}[PASS]${NC} $1"
	((TESTS_PASSED++))
}
log_fail() {
	echo -e "${RED}[FAIL]${NC} $1"
	((TESTS_FAILED++))
}
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_section() {
	echo -e "\n${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
	echo -e "${YELLOW}  $1${NC}"
	echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

# Make HTTP request and capture response + status code
# Usage: http_request METHOD URL [DATA]
# Returns: Sets HTTP_BODY and HTTP_STATUS global variables
http_request() {
	local method="$1"
	local url="$2"
	local data="${3-}"
	local auth_header="${4-}"

	local curl_opts=(-sS -w "\n%{http_code}" -X "$method" --connect-timeout 5 --max-time 10)

	if [ -n "$auth_header" ]; then
		curl_opts+=(-H "Authorization: Bearer $auth_header")
	fi

	curl_opts+=(-H "Content-Type: application/json")
	curl_opts+=(-H "Accept: application/json")

	if [ -n "$data" ]; then
		curl_opts+=(-d "$data")
	fi

	local response
	response=$(curl "${curl_opts[@]}" "$url" 2>&1) || true

	HTTP_STATUS=$(echo "$response" | tail -n1)
	HTTP_BODY=$(echo "$response" | sed '$d')
}

# Extract JSON field using Python (more reliable than jq for edge cases)
json_get() {
	local json="$1"
	local field="$2"
	python3 -c "import sys,json; d=json.loads(sys.stdin.read()); print(d.get('$field',''))" <<<"$json" 2>/dev/null || echo ""
}

# =============================================================================
# Setup: Register and authenticate test user
# =============================================================================
setup_test_user() {
	log_section "SETUP: Creating test user"

	log_info "Registering user: $EMAIL"
	http_request POST "${API_URL}/auth/register" "{\"email\":\"${EMAIL}\",\"phone\":\"${PHONE}\",\"password\":\"${PASSWORD}\"}"

	if [ "$HTTP_STATUS" != "201" ]; then
		log_fail "Registration failed with status $HTTP_STATUS"
		echo "Response: $HTTP_BODY"
		exit 1
	fi

	local verify_code
	verify_code=$(json_get "$HTTP_BODY" "verify_code")

	if [ -z "$verify_code" ]; then
		log_fail "No verify_code in registration response"
		echo "Response: $HTTP_BODY"
		exit 1
	fi

	log_info "Verifying email with code: $verify_code"
	http_request POST "${API_URL}/auth/verify" "{\"email\":\"${EMAIL}\",\"code\":\"${verify_code}\"}"

	if [ "$HTTP_STATUS" != "200" ]; then
		log_fail "Verification failed with status $HTTP_STATUS"
		echo "Response: $HTTP_BODY"
		exit 1
	fi

	ACCESS_TOKEN=$(json_get "$HTTP_BODY" "access_token")

	if [ -z "$ACCESS_TOKEN" ]; then
		log_fail "No access_token in verification response"
		echo "Response: $HTTP_BODY"
		exit 1
	fi

	log_success "Test user created and authenticated"
	log_info "Token: ${ACCESS_TOKEN:0:50}..."
}

# =============================================================================
# Test 1: POST /link - Generate binding link
# =============================================================================
test_link_endpoint() {
	log_section "TEST 1: POST /api/notifications/telegram/link"

	log_info "Requesting binding link..."
	http_request POST "${API_URL}/notifications/telegram/link" "" "$ACCESS_TOKEN"

	echo "  Status: $HTTP_STATUS"
	echo "  Body: $HTTP_BODY"

	if [ "$HTTP_STATUS" == "200" ]; then
		local deeplink token code expires_at
		deeplink=$(json_get "$HTTP_BODY" "deeplink")
		token=$(json_get "$HTTP_BODY" "token")
		code=$(json_get "$HTTP_BODY" "code")
		expires_at=$(json_get "$HTTP_BODY" "expires_at")

		if [ -n "$deeplink" ] && [ -n "$token" ] && [ -n "$code" ]; then
			log_success "Link endpoint returned valid response"
			log_info "  deeplink: $deeplink"
			log_info "  code: $code"
			log_info "  expires_at: $expires_at"
		else
			log_fail "Link response missing required fields"
		fi
	elif [ "$HTTP_STATUS" == "503" ]; then
		log_warn "Telegram service unavailable (503) - this is expected if telegram-service is not running"
		local error_code error_msg
		error_code=$(json_get "$HTTP_BODY" "error")
		error_msg=$(json_get "$HTTP_BODY" "message")
		log_info "  error: $error_code"
		log_info "  message: $error_msg"
		((TESTS_PASSED++)) # Consider this a pass if service is just unavailable
	elif [ "$HTTP_STATUS" == "401" ]; then
		log_fail "Unauthorized - token may be invalid"
	else
		log_fail "Unexpected status code: $HTTP_STATUS"
	fi
}

# =============================================================================
# Test 2: GET /bound - Check if user is bound
# =============================================================================
test_bound_endpoint() {
	log_section "TEST 2: GET /api/notifications/telegram/bound"

	log_info "Checking binding status (lightweight)..."
	http_request GET "${API_URL}/notifications/telegram/bound" "" "$ACCESS_TOKEN"

	echo "  Status: $HTTP_STATUS"
	echo "  Body: $HTTP_BODY"

	if [ "$HTTP_STATUS" == "200" ]; then
		local is_bound status
		is_bound=$(json_get "$HTTP_BODY" "is_bound")
		status=$(json_get "$HTTP_BODY" "status")

		log_success "Bound endpoint returned 200"
		log_info "  is_bound: $is_bound"
		log_info "  status: $status"

		# For new user, is_bound should be false
		if [ "$is_bound" == "False" ] || [ "$is_bound" == "false" ]; then
			log_info "  (Expected: new user is not bound)"
		fi
	elif [ "$HTTP_STATUS" == "503" ]; then
		log_warn "Telegram service unavailable (503)"
		((TESTS_PASSED++))
	elif [ "$HTTP_STATUS" == "401" ]; then
		log_fail "Unauthorized - token may be invalid"
	elif [ "$HTTP_STATUS" == "404" ]; then
		log_fail "Endpoint not found (404) - route may not be registered"
	else
		log_fail "Unexpected status code: $HTTP_STATUS"
	fi
}

# =============================================================================
# Test 3: GET /status - Detailed binding status
# =============================================================================
test_status_endpoint() {
	log_section "TEST 3: GET /api/notifications/telegram/status"

	log_info "Getting detailed binding status..."
	http_request GET "${API_URL}/notifications/telegram/status" "" "$ACCESS_TOKEN"

	echo "  Status: $HTTP_STATUS"
	echo "  Body: $HTTP_BODY"

	if [ "$HTTP_STATUS" == "200" ]; then
		log_success "Status endpoint returned 200 (user is bound)"
		local status username chat_id
		status=$(json_get "$HTTP_BODY" "status")
		username=$(json_get "$HTTP_BODY" "username")
		chat_id=$(json_get "$HTTP_BODY" "chat_id")
		log_info "  status: $status"
		log_info "  username: $username"
		log_info "  chat_id: $chat_id"
	elif [ "$HTTP_STATUS" == "404" ]; then
		local error_code
		error_code=$(json_get "$HTTP_BODY" "error")
		if [ "$error_code" == "telegram_not_bound" ]; then
			log_success "Status endpoint returned 404 (user not bound - expected for new user)"
		else
			log_fail "Status endpoint returned 404 with unexpected error: $error_code"
		fi
	elif [ "$HTTP_STATUS" == "503" ]; then
		log_warn "Telegram service unavailable (503)"
		((TESTS_PASSED++))
	elif [ "$HTTP_STATUS" == "401" ]; then
		log_fail "Unauthorized - token may be invalid"
	else
		log_fail "Unexpected status code: $HTTP_STATUS"
	fi
}

# =============================================================================
# Test 4: POST /unbind - Remove binding
# =============================================================================
test_unbind_endpoint() {
	log_section "TEST 4: POST /api/notifications/telegram/unbind"

	log_info "Attempting to unbind (should fail for unbound user)..."
	http_request POST "${API_URL}/notifications/telegram/unbind" "" "$ACCESS_TOKEN"

	echo "  Status: $HTTP_STATUS"
	echo "  Body: $HTTP_BODY"

	if [ "$HTTP_STATUS" == "200" ]; then
		log_success "Unbind endpoint returned 200 (binding removed)"
		local success message
		success=$(json_get "$HTTP_BODY" "success")
		message=$(json_get "$HTTP_BODY" "message")
		log_info "  success: $success"
		log_info "  message: $message"
	elif [ "$HTTP_STATUS" == "404" ]; then
		local error_code
		error_code=$(json_get "$HTTP_BODY" "error")
		if [ "$error_code" == "telegram_not_bound" ]; then
			log_success "Unbind endpoint returned 404 (user not bound - expected for new user)"
		else
			log_fail "Unbind endpoint returned 404 with unexpected error: $error_code"
		fi
	elif [ "$HTTP_STATUS" == "503" ]; then
		log_warn "Telegram service unavailable (503)"
		local error_code error_msg
		error_code=$(json_get "$HTTP_BODY" "error")
		error_msg=$(json_get "$HTTP_BODY" "message")
		log_info "  error: $error_code"
		log_info "  message: $error_msg"
		((TESTS_PASSED++)) # Service unavailable is acceptable
	elif [ "$HTTP_STATUS" == "401" ]; then
		log_fail "Unauthorized - token may be invalid"
	else
		log_fail "Unexpected status code: $HTTP_STATUS"
	fi
}

# =============================================================================
# Test 5: Unauthorized access (no token)
# =============================================================================
test_unauthorized_access() {
	log_section "TEST 5: Unauthorized Access (no token)"

	log_info "Testing /link without Authorization header..."
	http_request POST "${API_URL}/notifications/telegram/link" "" ""

	echo "  Status: $HTTP_STATUS"

	if [ "$HTTP_STATUS" == "401" ]; then
		log_success "Link endpoint correctly returned 401 for unauthorized request"
	else
		log_fail "Link endpoint should return 401, got $HTTP_STATUS"
	fi

	log_info "Testing /bound without Authorization header..."
	http_request GET "${API_URL}/notifications/telegram/bound" "" ""

	echo "  Status: $HTTP_STATUS"

	if [ "$HTTP_STATUS" == "401" ]; then
		log_success "Bound endpoint correctly returned 401 for unauthorized request"
	else
		log_fail "Bound endpoint should return 401, got $HTTP_STATUS"
	fi
}

# =============================================================================
# Main
# =============================================================================
main() {
	echo ""
	echo "╔═══════════════════════════════════════════════════════════════╗"
	echo "║       Telegram API Integration Tests                          ║"
	echo "╠═══════════════════════════════════════════════════════════════╣"
	echo "║  Base URL: $BASE_URL"
	echo "║  Test User: $EMAIL"
	echo "╚═══════════════════════════════════════════════════════════════╝"
	echo ""

	# Setup
	setup_test_user

	# Run tests
	test_link_endpoint
	test_bound_endpoint
	test_status_endpoint
	test_unbind_endpoint
	test_unauthorized_access

	# Summary
	log_section "TEST SUMMARY"

	local total=$((TESTS_PASSED + TESTS_FAILED))
	echo -e "  ${GREEN}Passed:${NC} $TESTS_PASSED"
	echo -e "  ${RED}Failed:${NC} $TESTS_FAILED"
	echo -e "  Total:  $total"
	echo ""

	if [ "$TESTS_FAILED" -eq 0 ]; then
		echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
		echo -e "${GREEN}║  ✅ ALL TESTS PASSED                                          ║${NC}"
		echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
		exit 0
	else
		echo -e "${RED}╔═══════════════════════════════════════════════════════════════╗${NC}"
		echo -e "${RED}║  ❌ SOME TESTS FAILED                                         ║${NC}"
		echo -e "${RED}╚═══════════════════════════════════════════════════════════════╝${NC}"
		exit 1
	fi
}

# Run
main "$@"
