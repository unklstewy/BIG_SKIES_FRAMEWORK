#!/bin/bash
# install-tools.sh - Install development tools for BIG SKIES Framework

set -e

echo "Installing development tools for BIG SKIES Framework..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go first."
    exit 1
fi

echo "Go version: $(go version)"

# Install golangci-lint
echo "Installing golangci-lint..."
if ! command -v golangci-lint &> /dev/null; then
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
else
    echo "golangci-lint already installed"
fi

# Install goimports
echo "Installing goimports..."
if ! command -v goimports &> /dev/null; then
    go install golang.org/x/tools/cmd/goimports@latest
else
    echo "goimports already installed"
fi

# Install staticcheck
echo "Installing staticcheck..."
if ! command -v staticcheck &> /dev/null; then
    go install honnef.co/go/tools/cmd/staticcheck@latest
else
    echo "staticcheck already installed"
fi

# Install gotestsum for better test output
echo "Installing gotestsum..."
if ! command -v gotestsum &> /dev/null; then
    go install gotest.tools/gotestsum@latest
else
    echo "gotestsum already installed"
fi

echo ""
echo "✅ Development tools installed successfully!"
echo ""
echo "Available tools:"
echo "  - golangci-lint: $(command -v golangci-lint || echo 'not found')"
echo "  - goimports: $(command -v goimports || echo 'not found')"
echo "  - staticcheck: $(command -v staticcheck || echo 'not found')"
echo "  - gotestsum: $(command -v gotestsum || echo 'not found')"
echo ""
echo "Next steps:"
echo "  1. Run 'make fmt' to format code"
echo "  2. Run 'make lint' to check code quality"
echo "  3. Run 'make test' to run tests"
