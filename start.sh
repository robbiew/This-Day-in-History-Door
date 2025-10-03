#!/bin/bash
# Wrapper script to start the Day-in-History door with sensible defaults
# Usage: ./start.sh /sbbs/node1

set -e

# Change to application directory (adjust if installed elsewhere)
cd /sbbs/xtrn/history

# First argument is path to node/dropfile
NODE_PATH="${1:-/sbbs/node1}"

# Default strategy and shuffle (shuffle enabled)
STRATEGY="era-based"
SHUFFLE_FLAG="-shuffle"

# Run the binary with chosen flags
./history -path "$NODE_PATH" $SHUFFLE_FLAG -strategy="$STRATEGY"
