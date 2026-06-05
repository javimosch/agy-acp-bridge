#!/bin/bash
set -e

cd "$(dirname "$0")"

echo "Building agy-acp-bridge..."
go build -ldflags "-s -w" -o agy-acp-bridge .
echo "Done: $(ls -lh agy-acp-bridge | awk '{print $5, $NF}')"
