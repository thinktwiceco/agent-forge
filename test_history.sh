#!/bin/bash

# Test script to verify conversation history retention

echo "Testing conversation history retention..."
echo ""

# Create a unique marker to identify this test conversation
MARKER="TestMarker_$(date +%s)"

# Start the agent with two messages
echo "Test 1: First message mentions: $MARKER" | ./thinktwice-agent 2>&1 | grep -A5 "Agent is thinking"

echo ""
echo "======================================"
echo ""

echo "Test 2: What did I say in the first message?" | ./thinktwice-agent 2>&1 | grep -A10 "Agent is thinking"

echo ""
echo "Test complete. Check if the agent remembered the marker."
