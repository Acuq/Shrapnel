#!/bin/bash

# Build script for Shrapnel Multi-IP Proxy Manager

set -e

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_DIR"

echo "Building Shrapnel Multi-IP Proxy Manager..."

# Set up local Go workspace for development
export GOPATH="$PROJECT_DIR/.go"
export GOBIN="$PROJECT_DIR/bin"
mkdir -p "$GOPATH/bin"

# Initialize go work for monorepo
echo "Setting up Go workspace..."
go work init
go work use ./cmd/manager
go work use ./pkg/profile
go work use ./pkg/config
go work use ./pkg/service

# Build manager with local packages
echo "Building manager..."
cd cmd/manager
go mod tidy
go build -o "$PROJECT_DIR/shrapnel-manager" .

cd "$PROJECT_DIR"

echo "Build completed successfully!"
echo "Binary: $PROJECT_DIR/shrapnel-manager"