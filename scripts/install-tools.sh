#!/bin/bash
# install-tools.sh - Install development tools for BIG SKIES Framework
# Supports: Linux (RPM, DEB, Arch), macOS (Homebrew), Windows (Git Bash/WSL), BSD

set -e

echo "BIG SKIES Framework - Development Tools Installation"
echo "====================================================="
echo ""

# Detect OS and distribution
detect_os() {
    OS="unknown"
    DISTRO="unknown"
    PACKAGE_MANAGER="none"
    
    # Detect OS
    case "$(uname -s)" in
        Linux*)
            OS="linux"
            # Detect Linux distribution
            if [ -f /etc/os-release ]; then
                . /etc/os-release
                DISTRO="$ID"
                case "$ID" in
                    ubuntu|debian|linuxmint|pop)
                        PACKAGE_MANAGER="apt"
                        ;;
                    fedora|rhel|centos|rocky|almalinux)
                        PACKAGE_MANAGER="dnf"
                        if ! command -v dnf &> /dev/null && command -v yum &> /dev/null; then
                            PACKAGE_MANAGER="yum"
                        fi
                        ;;
                    arch|manjaro|endeavouros)
                        PACKAGE_MANAGER="pacman"
                        ;;
                    opensuse*|sles)
                        PACKAGE_MANAGER="zypper"
                        ;;
                    alpine)
                        PACKAGE_MANAGER="apk"
                        ;;
                esac
            fi
            ;;
        Darwin*)
            OS="macos"
            if command -v brew &> /dev/null; then
                PACKAGE_MANAGER="brew"
            fi
            ;;
        CYGWIN*|MINGW*|MSYS*)
            OS="windows"
            if command -v choco &> /dev/null; then
                PACKAGE_MANAGER="choco"
            elif command -v scoop &> /dev/null; then
                PACKAGE_MANAGER="scoop"
            fi
            ;;
        FreeBSD*)
            OS="freebsd"
            PACKAGE_MANAGER="pkg"
            ;;
        OpenBSD*)
            OS="openbsd"
            PACKAGE_MANAGER="pkg_add"
            ;;
        NetBSD*)
            OS="netbsd"
            PACKAGE_MANAGER="pkgin"
            ;;
    esac
    
    echo "Detected OS: $OS"
    if [ "$DISTRO" != "unknown" ]; then
        echo "Distribution: $DISTRO"
    fi
    echo "Package Manager: $PACKAGE_MANAGER"
    echo ""
}

# Install Go if not present
install_go() {
    if command -v go &> /dev/null; then
        echo "✓ Go already installed: $(go version)"
        return 0
    fi
    
    echo "Go is not installed. Installing..."
    
    case "$PACKAGE_MANAGER" in
        apt)
            echo "Installing via apt..."
            sudo apt-get update
            sudo apt-get install -y golang-go
            ;;
        dnf|yum)
            echo "Installing via $PACKAGE_MANAGER..."
            sudo $PACKAGE_MANAGER install -y golang
            ;;
        pacman)
            echo "Installing via pacman..."
            sudo pacman -S --noconfirm go
            ;;
        zypper)
            echo "Installing via zypper..."
            sudo zypper install -y go
            ;;
        apk)
            echo "Installing via apk..."
            sudo apk add go
            ;;
        brew)
            echo "Installing via Homebrew..."
            brew install go
            ;;
        pkg)
            echo "Installing via pkg (FreeBSD)..."
            sudo pkg install -y go
            ;;
        pkg_add)
            echo "Installing via pkg_add (OpenBSD)..."
            sudo pkg_add go
            ;;
        pkgin)
            echo "Installing via pkgin (NetBSD)..."
            sudo pkgin -y install go
            ;;
        choco)
            echo "Installing via Chocolatey..."
            choco install -y golang
            ;;
        scoop)
            echo "Installing via Scoop..."
            scoop install go
            ;;
        *)
            echo "⚠️  Cannot auto-install Go on this system."
            echo "Please install Go manually from: https://go.dev/dl/"
            echo "Then run this script again."
            return 1
            ;;
    esac
    
    # Verify installation
    if command -v go &> /dev/null; then
        echo "✓ Go installed successfully: $(go version)"
        return 0
    else
        echo "❌ Go installation failed. Please install manually."
        return 1
    fi
}

# Check if Go is installed
detect_os

if ! command -v go &> /dev/null; then
    read -p "Go is not installed. Install it now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        install_go || exit 1
    else
        echo "❌ Go is required. Exiting."
        exit 1
    fi
fi

echo "Go version: $(go version)"
echo "GOPATH: ${GOPATH:-$HOME/go}"
echo ""

# Ensure GOPATH/bin is in PATH
GOPATH_BIN="${GOPATH:-$HOME/go}/bin"
if [[ ":$PATH:" != *":$GOPATH_BIN:"* ]]; then
    echo "⚠️  Warning: $GOPATH_BIN is not in PATH"
    echo "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo "  export PATH=\$PATH:$GOPATH_BIN"
    echo ""
    # Temporarily add to PATH for this session
    export PATH="$PATH:$GOPATH_BIN"
fi

# Function to install a Go tool
install_go_tool() {
    local tool_name="$1"
    local tool_package="$2"
    local tool_cmd="${3:-$tool_name}"  # Command name (may differ from package name)
    
    echo "Installing $tool_name..."
    if command -v "$tool_cmd" &> /dev/null; then
        local version=$("$tool_cmd" version 2>/dev/null || "$tool_cmd" --version 2>/dev/null || echo "unknown")
        echo "  ✓ Already installed: $version"
        return 0
    fi
    
    echo "  Downloading and building $tool_package..."
    if go install "$tool_package"@latest; then
        echo "  ✓ Installed successfully"
        return 0
    else
        echo "  ❌ Failed to install $tool_name"
        return 1
    fi
}

echo "Installing Go development tools..."
echo ""

# Install golangci-lint
install_go_tool "golangci-lint" "github.com/golangci/golangci-lint/cmd/golangci-lint" || true

# Install goimports
install_go_tool "goimports" "golang.org/x/tools/cmd/goimports" || true

# Install staticcheck
install_go_tool "staticcheck" "honnef.co/go/tools/cmd/staticcheck" || true

# Install gotestsum for better test output
install_go_tool "gotestsum" "gotest.tools/gotestsum" || true

# Install govulncheck for security scanning
install_go_tool "govulncheck" "golang.org/x/vuln/cmd/govulncheck" || true

echo ""
echo "======================================"
echo "✅ Development tools installation complete!"
echo "======================================"
echo ""
echo "Installed tools:"

# Check each tool and show status
check_tool() {
    local tool="$1"
    if command -v "$tool" &> /dev/null; then
        local path=$(command -v "$tool")
        echo "  ✓ $tool: $path"
        return 0
    else
        echo "  ✗ $tool: not found"
        return 1
    fi
}

check_tool "golangci-lint"
check_tool "goimports"
check_tool "staticcheck"
check_tool "gotestsum"
check_tool "govulncheck"

echo ""
echo "Tool versions:"
command -v golangci-lint &> /dev/null && echo "  golangci-lint: $(golangci-lint version 2>&1 | head -1)"
command -v staticcheck &> /dev/null && echo "  staticcheck: $(staticcheck -version 2>&1)"
command -v gotestsum &> /dev/null && echo "  gotestsum: $(gotestsum --version 2>&1)"

echo ""
echo "PATH configuration:"
echo "  GOPATH: ${GOPATH:-$HOME/go}"
echo "  GOBIN: $GOPATH_BIN"
if [[ ":$PATH:" == *":$GOPATH_BIN:"* ]]; then
    echo "  ✓ GOPATH/bin is in PATH"
else
    echo "  ⚠️  GOPATH/bin is NOT in PATH (temporarily added for this session)"
    echo "  Add to your shell profile:"
    echo "    export PATH=\$PATH:$GOPATH_BIN"
fi

echo ""
echo "Next steps:"
echo "  1. Run 'make fmt' to format code"
echo "  2. Run 'make lint' to check code quality"
echo "  3. Run 'make test' to run tests"
echo "  4. Run 'go mod tidy' to clean up dependencies"
echo ""
echo "For more information:"
echo "  - golangci-lint: https://golangci-lint.run/"
echo "  - staticcheck: https://staticcheck.io/"
echo "  - goimports: https://pkg.go.dev/golang.org/x/tools/cmd/goimports"
echo "  - govulncheck: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck"
