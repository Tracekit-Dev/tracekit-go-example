#!/bin/bash

echo "🧪 TraceKit Code Monitoring - End-to-End Test"
echo "=============================================="
echo ""

BASE_URL="http://localhost:8082"
BACKEND_URL="http://localhost:8081"

echo "📊 Test 1: Generate traces with stack traces (trigger code discovery)"
echo "----------------------------------------------------------------------"
echo "Hitting /api/error endpoint to generate errors with stack traces..."
for i in {1..5}; do
  response=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/error")
  http_code=$(echo "$response" | tail -n1)
  echo "  Request $i: HTTP $http_code"
  sleep 0.5
done
echo ""

echo "⏳ Waiting 3 seconds for traces to be processed..."
sleep 3
echo ""

echo "📈 Test 2: Generate regular traffic"
echo "-----------------------------------"
echo "Hitting various endpoints to generate spans..."
curl -s "$BASE_URL/" > /dev/null && echo "  ✓ GET /"
curl -s "$BASE_URL/api/users?limit=10" > /dev/null && echo "  ✓ GET /api/users"
curl -s "$BASE_URL/api/order?amount=500&items=3" > /dev/null && echo "  ✓ GET /api/order (amount=500)"
curl -s "$BASE_URL/api/order?amount=1500&items=5" > /dev/null && echo "  ✓ GET /api/order (amount=1500)"
curl -s "$BASE_URL/api/order?amount=2000&items=7" > /dev/null && echo "  ✓ GET /api/order (amount=2000)"
echo ""

echo "⏳ Waiting 2 seconds for traces to be ingested..."
sleep 2
echo ""

echo "📋 Test 3: Check discovered code"
echo "--------------------------------"
echo "You should now see code locations in the UI at:"
echo "  👉 $BACKEND_URL/snapshots (Browse Code tab)"
echo ""
echo "Expected discoveries:"
echo "  • service_name: test-app"
echo "  • file_path: main.go"
echo "  • functions: main.errorHandler, main.orderHandler, etc."
echo ""

echo "🎯 Test 4: Create a breakpoint"
echo "------------------------------"
echo "Steps:"
echo "  1. Go to: $BACKEND_URL/snapshots"
echo "  2. Click 'Browse Code' tab"
echo "  3. Find 'test-app' service -> 'main.go'"
echo "  4. Click '🎯 Set Breakpoint' on line 120 (orderHandler)"
echo "  5. Modal should auto-fill:"
echo "     - Service: test-app"
echo "     - File: main.go"
echo "     - Line: 120"
echo "     - Function: main.orderHandler"
echo "  6. Optionally add condition: amount > 1000"
echo "  7. Click 'Create Breakpoint'"
echo ""

read -p "Press ENTER after creating the breakpoint..."
echo ""

echo "🚀 Test 5: Trigger breakpoint captures"
echo "--------------------------------------"
echo "Sending 50 requests to trigger the breakpoint..."
for i in {1..50}; do
  amount=$((500 + RANDOM % 2000))
  curl -s "$BASE_URL/api/order?amount=$amount&items=3" > /dev/null
  if [ $((i % 10)) -eq 0 ]; then
    echo "  Sent $i requests..."
  fi
  sleep 0.1
done
echo "  ✅ Sent 50 requests with varying amounts"
echo ""

echo "⏳ Waiting 3 seconds for snapshots to be captured..."
sleep 3
echo ""

echo "👁️ Test 6: View captured snapshots"
echo "----------------------------------"
echo "Steps:"
echo "  1. Go to: $BACKEND_URL/snapshots (Breakpoints tab)"
echo "  2. You should see your breakpoint with captures"
echo "  3. Click '👁️ View Snapshots'"
echo "  4. Verify you see captured data:"
echo "     - Local variables (amount, items, etc.)"
echo "     - Parameters"
echo "     - Stack traces"
echo "     - Request context"
echo ""

read -p "Press ENTER to continue..."
echo ""

echo "🧹 Test 7: Service filter synchronization"
echo "----------------------------------------"
echo "Steps:"
echo "  1. Select 'test-app' from service dropdown"
echo "  2. Switch between 'Breakpoints' and 'Browse Code' tabs"
echo "  3. Verify filter persists across tabs"
echo "  4. Reload page - filter should remain selected"
echo ""

read -p "Press ENTER to continue..."
echo ""

echo "📊 Final traffic generation"
echo "--------------------------"
echo "Generating final burst of traffic..."
for i in {1..20}; do
  curl -s "$BASE_URL/api/users?limit=5" > /dev/null
  curl -s "$BASE_URL/api/order?amount=1200&items=4" > /dev/null
  sleep 0.2
done
echo "  ✅ Generated 40 more requests"
echo ""

echo "✅ End-to-End Test Complete!"
echo "============================"
echo ""
echo "Summary:"
echo "  • Test app running on: $BASE_URL"
echo "  • TraceKit backend: $BACKEND_URL"
echo "  • Service name: test-app"
echo ""
echo "Next steps:"
echo "  1. Check captured snapshots: $BACKEND_URL/snapshots"
echo "  2. Verify code discovery worked"
echo "  3. Test breakpoint toggle (enable/disable)"
echo "  4. Test breakpoint deletion"
echo "  5. Verify trace links work from snapshot view"
echo ""
echo "📝 Check app logs:"
echo "  tail -f $(pwd)/app.log"
echo ""

