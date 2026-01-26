#!/bin/bash

# Test script for the new Opencode-compatible /prompt endpoint

BASE_URL="${BASE_URL:-http://localhost:8080}"
PROJECT_DIR="${PROJECT_DIR:-/tmp/test_project}"

echo "=== Testing Zorkagent Opencode-compatible API ==="
echo "Base URL: $BASE_URL"
echo "Project Directory: $PROJECT_DIR"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test 1: Create a session
echo "Test 1: Creating a session..."
SESSION_RESPONSE=$(curl -s -X POST "${BASE_URL}/session?directory=${PROJECT_DIR}" \
  -H "Content-Type: application/json")

SESSION_ID=$(echo "$SESSION_RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4)

if [ -n "$SESSION_ID" ]; then
    echo -e "${GREEN}✓ Session created successfully${NC}"
    echo "  Session ID: $SESSION_ID"
else
    echo -e "${RED}✗ Failed to create session${NC}"
    echo "  Response: $SESSION_RESPONSE"
    exit 1
fi
echo ""

# Test 2: Send a message using the new /prompt endpoint
echo "Test 2: Sending a message using /prompt endpoint..."
PROMPT_RESPONSE=$(curl -s -X POST "${BASE_URL}/session/${SESSION_ID}/prompt?directory=${PROJECT_DIR}" \
  -H "Content-Type: application/json" \
  -d '{
    "parts": [
      {
        "text": "Hello, please respond with just \"API test successful\""
      }
    ]
  }')

echo "Response:"
echo "$PROMPT_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$PROMPT_RESPONSE"

# Check if response contains expected fields
if echo "$PROMPT_RESPONSE" | grep -q '"info"' && echo "$PROMPT_RESPONSE" | grep -q '"parts"'; then
    echo -e "\n${GREEN}✓ Prompt endpoint works correctly${NC}"
else
    echo -e "\n${RED}✗ Prompt endpoint response format error${NC}"
fi
echo ""

# Test 3: Test with model specification
echo "Test 3: Testing with model specification..."
MODEL_RESPONSE=$(curl -s -X POST "${BASE_URL}/session/${SESSION_ID}/prompt?directory=${PROJECT_DIR}" \
  -H "Content-Type: application/json" \
  -d '{
    "model": {
      "providerID": "anthropic",
      "modelID": "claude-sonnet-4-20250514"
    },
    "parts": [
      {
        "text": "What is 2+2? Answer with just the number."
      }
    ]
  }')

echo "Response:"
echo "$MODEL_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$MODEL_RESPONSE"
echo ""

# Test 4: Test noReply mode
echo "Test 4: Testing noReply mode..."
NOREPLY_RESPONSE=$(curl -s -X POST "${BASE_URL}/session/${SESSION_ID}/prompt?directory=${PROJECT_DIR}" \
  -H "Content-Type: application/json" \
  -d '{
    "noReply": true,
    "parts": [
      {
        "text": "This message should not get a response"
      }
    ]
  }')

echo "Response:"
echo "$NOREPLY_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$NOREPLY_RESPONSE"

if echo "$NOREPLY_RESPONSE" | grep -q '"role":"user"'; then
    echo -e "\n${GREEN}✓ NoReply mode works correctly${NC}"
else
    echo -e "\n${RED}✗ NoReply mode failed${NC}"
fi
echo ""

# Test 5: Verify the old /message endpoint is gone
echo "Test 5: Verifying old /message endpoint is replaced..."
OLD_ENDPOINT_RESPONSE=$(curl -s -X POST "${BASE_URL}/session/${SESSION_ID}/message?directory=${PROJECT_DIR}" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "test",
    "stream": false
  }')

if echo "$OLD_ENDPOINT_RESPONSE" | grep -q '404\|405\|not found'; then
    echo -e "${GREEN}✓ Old /message endpoint properly removed${NC}"
else
    echo -e "${RED}⚠ Old /message endpoint still responds (expected if using parallel endpoints strategy)${NC}"
fi
echo ""

echo "=== All tests completed ==="
