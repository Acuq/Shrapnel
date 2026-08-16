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

# Build manager with local packages
echo "Building manager..."
cd cmd/manager

# Replace local imports in go.mod
go mod edit -replace github.com/Acuq/shrapnel/pkg/profile="$PROJECT_DIR/pkg/profile"
go mod edit -replace github.com/Acuq/shrapnel/pkg/config="$PROJECT_DIR/pkg/config"
go mod edit -replace github.com/Acuq/shrapnel/pkg/service="$PROJECT_DIR/pkg/service"

# Tidy and build
go mod tidy
go build -o "$PROJECT_DIR/shrapnel-manager" .

cd "$PROJECT_DIR"

echo "Build completed successfully!"
echo "Binary: $PROJECT_DIR/shrapnel-manager"