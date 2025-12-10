#!/bin/bash

# Simple test script for TraceKit Go SDK

echo "🧪 Testing TraceKit Go SDK"
echo "================================"
echo ""

BASE_URL="http://localhost:8082"

echo "1️⃣  Testing Hello Endpoint..."
curl -s $BASE_URL/ | jq -r '.message' 2>/dev/null || curl -s $BASE_URL/
echo ""

echo ""
echo "2️⃣  Fetching Users (5 times)..."
for i in {1..5}; do
  curl -s $BASE_URL/api/users > /dev/null && echo "  ✓ Request $i"
done

echo ""
echo "3️⃣  Creating Orders (3 times)..."
for i in {1..3}; do
  ORDER=$(curl -s -X POST $BASE_URL/api/order)
  ORDER_ID=$(echo $ORDER | jq -r '.order_id' 2>/dev/null || echo "N/A")
  echo "  ✓ Created: $ORDER_ID"
done

echo ""
echo "4️⃣  Triggering Error..."
curl -s $BASE_URL/api/error | jq -r '.error' 2>/dev/null || curl -s $BASE_URL/api/error
echo ""

echo ""
echo "5️⃣  Health Check..."
curl -s $BASE_URL/health | jq . 2>/dev/null || curl -s $BASE_URL/health
echo ""

echo ""
echo "================================"
echo "✅ All tests completed!"
echo ""
echo "📊 View traces at: http://localhost:8081/traces"
echo "   Service name: test-app"
echo ""

