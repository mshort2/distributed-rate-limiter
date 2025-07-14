#!/bin/bash

echo "Starting load test..."
echo "Warming up server..."

# Warm up
wrk -t2 -c10 -d10s --script=test/load_test.lua http://localhost:8080/check

echo "Running main test..."

# Main test
wrk -t4 -c100 -d30s --script=test/load_test.lua http://localhost:8080/check

echo "Test complete!"
