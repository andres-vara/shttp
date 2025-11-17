#!/bin/bash
# test-auth.sh - Test the AuthMiddleware implementation

set -e

echo "Chi Multi-Provider Example - AuthMiddleware Test"
echo "=================================================="
echo ""

# Start the server in background
./chi-multi > /tmp/chi-server.log 2>&1 &
SERVER_PID=$!
trap "kill $SERVER_PID 2>/dev/null || true" EXIT

# Wait for server to start
sleep 2

BASE_URL="http://localhost:8080"

echo "Test 1: Request WITHOUT Authorization header (should get 401)"
echo "Command: curl -X GET $BASE_URL/providers/prv1/users"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/providers/prv1/users")
echo "Response: HTTP $HTTP_CODE"
if [ "$HTTP_CODE" = "401" ]; then
  echo "✓ PASS: Correctly rejected request without auth"
else
  echo "✗ FAIL: Expected 401, got $HTTP_CODE"
fi
echo ""

echo "Test 2: Request WITH invalid Authorization format (should get 401)"
echo "Command: curl -X GET $BASE_URL/providers/prv1/users -H 'Authorization: Invalid'"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/providers/prv1/users" \
  -H "Authorization: Invalid")
echo "Response: HTTP $HTTP_CODE"
if [ "$HTTP_CODE" = "401" ]; then
  echo "✓ PASS: Correctly rejected invalid auth format"
else
  echo "✗ FAIL: Expected 401, got $HTTP_CODE"
fi
echo ""

echo "Test 3: Request WITH valid token and valid provider (should succeed)"
TOKEN='{"client_ip":"192.168.1.100","request_id":"req-001"}'
echo "Command: curl -X POST $BASE_URL/providers/prv1/users \\"
echo "  -H 'Authorization: Bearer $TOKEN' \\"
echo "  -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\"}'"
HTTP_CODE=$(curl -s -o /tmp/response.json -w "%{http_code}" -X POST "$BASE_URL/providers/prv1/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Alice","email":"alice@example.com"}')
echo "Response: HTTP $HTTP_CODE"
if [ "$HTTP_CODE" = "201" ]; then
  echo "✓ PASS: Successfully created user with auth"
  echo "Response body:"
  cat /tmp/response.json | python3 -m json.tool 2>/dev/null || cat /tmp/response.json
else
  echo "✗ FAIL: Expected 201, got $HTTP_CODE"
fi
echo ""

echo "Test 4: Request with different provider using same auth (should succeed)"
TOKEN2='{"client_ip":"192.168.1.101","request_id":"req-002"}'
echo "Command: curl -X POST $BASE_URL/providers/prv2/users \\"
echo "  -H 'Authorization: Bearer $TOKEN2' \\"
echo "  -d '{\"name\":\"Bob\",\"email\":\"bob@example.com\"}'"
HTTP_CODE=$(curl -s -o /tmp/response2.json -w "%{http_code}" -X POST "$BASE_URL/providers/prv2/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN2" \
  -d '{"name":"Bob","email":"bob@example.com"}')
echo "Response: HTTP $HTTP_CODE"
if [ "$HTTP_CODE" = "201" ]; then
  echo "✓ PASS: Successfully created user with different provider"
  echo "Response body:"
  cat /tmp/response2.json | python3 -m json.tool 2>/dev/null || cat /tmp/response2.json
else
  echo "✗ FAIL: Expected 201, got $HTTP_CODE"
fi
echo ""

echo "Test 5: List users with auth"
echo "Command: curl -X GET $BASE_URL/providers/prv1/users \\"
echo "  -H 'Authorization: Bearer $TOKEN'"
HTTP_CODE=$(curl -s -o /tmp/response3.json -w "%{http_code}" -X GET "$BASE_URL/providers/prv1/users" \
  -H "Authorization: Bearer $TOKEN")
echo "Response: HTTP $HTTP_CODE"
if [ "$HTTP_CODE" = "200" ]; then
  echo "✓ PASS: Successfully listed users"
  echo "Response body:"
  cat /tmp/response3.json | python3 -m json.tool 2>/dev/null || cat /tmp/response3.json
else
  echo "✗ FAIL: Expected 200, got $HTTP_CODE"
fi
echo ""

echo "=================================================="
echo "All tests complete!"
echo "Server logs saved to: /tmp/chi-server.log"
echo ""
echo "To view logs with auth claims:"
echo "  cat /tmp/chi-server.log | grep -E '(auth_success|request_start|user_created)'"
