#!/bin/bash

# Generate a 32-byte random key and encode it as Base64
if command -v openssl >/dev/null 2>&1; then
    openssl rand -base64 32
elif command -v head >/dev/null 2>&1; then
    head -c 32 /dev/urandom | base64
else
    echo "Error: openssl or base64 not found. Please run the Go script: go run scripts/gen_key.go"
    exit 1
fi
