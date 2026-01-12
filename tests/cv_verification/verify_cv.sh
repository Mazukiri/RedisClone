#!/bin/bash
set -e

# Start Server
echo "Starting MemKV Server..."
../../memkv-server > /dev/null 2>&1 &
SERVER_PID=$!
sleep 2

echo "=============================================="
echo "TEST 1: Throughput Verification (50k+ ops/sec)"
echo "=============================================="
# Run redis-benchmark using parallel clients and pipelining to maximize throughput
# Using -t set,get to match CV verification claim
# Use -q for quiet output showing only req/sec
redis-benchmark -p 8082 -t set,get -n 100000 -c 50 -q

echo ""
echo "=============================================="
echo "TEST 2: C10k Connection Test"
echo "=============================================="
# Need to increase ulimit for this session to allow many open files
ulimit -n 65535 || echo "Warning: Could not set ulimit, C10k might fail if limit is low"
go run c10k_test.go

echo ""
echo "=============================================="
echo "TEST 3: Data Structure Complexity Verification"
echo "=============================================="
echo "Verifying SkipList O(log N) behavior..."
# Run the benchmark we created
# We are looking for Logarithmic growth in time per op, essentially time/op should increase slowly.
# Since we are just verifying, we run the benchmark and print it.
cd ../../
go test -bench=BenchmarkSkipListSearch -benchmem ./tests/cv_verification/...

# Cleanup
kill $SERVER_PID
