#!/bin/bash
# setup-dev-environment.sh - Complete development environment setup for BIG SKIES Framework
# Supports: Linux (RPM, DEB, Arch), macOS (Homebrew), Windows (WSL/Git Bash), BSD

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║  BIG SKIES Framework - Development Environment Setup          ║"
echo "║  This script will install all required development tools      ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

# Detect OS and distribution
detect_os() {
    OS="unknown"
    DISTRO="unknown"
    PACKAGE_MANAGER="none"
    
    case "$(uname -s)" in
        Linux*)
            OS="linux"
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
            PACKAGE_MANAGER="brew"
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
    
    log_info "Detected OS: $OS"
    [ "$DISTRO" != "unknown" ] && log_info "Distribution: $DISTRO"
    log_info "Package Manager: $PACKAGE_MANAGER"
    echo ""
}

# Check if running with sudo when needed
check_sudo() {
    if [ "$OS" = "linux" ] || [ "$OS" = "freebsd" ] || [ "$OS" = "openbsd" ] || [ "$OS" = "netbsd" ]; then
        if [ "$EUID" -ne 0 ] && ! sudo -n true 2>/dev/null; then
            log_warning "Some installations may require sudo privileges"
            log_info "You may be prompted for your password"
            echo ""
        fi
    fi
}

# Install package manager if needed (macOS Homebrew)
install_package_manager() {
    if [ "$OS" = "macos" ] && ! command -v brew &> /dev/null; then
        log_info "Homebrew not found. Installing Homebrew..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
        
        # Add Homebrew to PATH for Apple Silicon
        if [ -d "/opt/homebrew/bin" ]; then
            eval "$(/opt/homebrew/bin/brew shellenv)"
        fi
        
        if command -v brew &> /dev/null; then
            log_success "Homebrew installed successfully"
            PACKAGE_MANAGER="brew"
        else
            log_error "Homebrew installation failed"
            return 1
        fi
    fi
}

# Install a package using the system package manager
install_package() {
    local package_name="$1"
    local package_map_apt="$2"
    local package_map_dnf="$3"
    local package_map_pacman="$4"
    local package_map_brew="$5"
    
    case "$PACKAGE_MANAGER" in
        apt)
            sudo apt-get update -qq
            sudo apt-get install -y "$package_map_apt"
            ;;
        dnf|yum)
            sudo $PACKAGE_MANAGER install -y "$package_map_dnf"
            ;;
        pacman)
            sudo pacman -S --noconfirm "$package_map_pacman"
            ;;
        zypper)
            sudo zypper install -y "$package_name"
            ;;
        apk)
            sudo apk add "$package_name"
            ;;
        brew)
            brew install "$package_map_brew"
            ;;
        pkg)
            sudo pkg install -y "$package_name"
            ;;
        pkg_add)
            sudo pkg_add "$package_name"
            ;;
        pkgin)
            sudo pkgin -y install "$package_name"
            ;;
        choco)
            choco install -y "$package_name"
            ;;
        scoop)
            scoop install "$package_name"
            ;;
        *)
            log_warning "Cannot auto-install $package_name on this system"
            return 1
            ;;
    esac
}

# Check and install Git
install_git() {
    log_info "Checking Git..."
    if command -v git &> /dev/null; then
        log_success "Git already installed: $(git --version)"
    else
        log_info "Installing Git..."
        install_package "git" "git" "git" "git" "git"
        if command -v git &> /dev/null; then
            log_success "Git installed: $(git --version)"
        else
            log_error "Git installation failed"
            return 1
        fi
    fi
}

# Check and install Make
install_make() {
    log_info "Checking Make..."
    if command -v make &> /dev/null; then
        log_success "Make already installed: $(make --version | head -1)"
    else
        log_info "Installing Make..."
        case "$OS" in
            linux)
                install_package "make" "build-essential" "make" "base-devel" ""
                ;;
            macos)
                # Xcode Command Line Tools includes make
                if ! xcode-select -p &> /dev/null; then
                    log_info "Installing Xcode Command Line Tools..."
                    xcode-select --install
                    log_warning "Please complete Xcode installation and run this script again"
                    exit 0
                fi
                ;;
            *)
                install_package "make" "make" "make" "make" "make"
                ;;
        esac
        
        if command -v make &> /dev/null; then
            log_success "Make installed: $(make --version | head -1)"
        else
            log_error "Make installation failed"
            return 1
        fi
    fi
}

# Check and install Go
install_go() {
    log_info "Checking Go..."
    if command -v go &> /dev/null; then
        log_success "Go already installed: $(go version)"
    else
        log_info "Installing Go..."
        install_package "golang" "golang-go" "golang" "go" "go"
        
        # Verify and set up Go environment
        if command -v go &> /dev/null; then
            log_success "Go installed: $(go version)"
            
            # Set up GOPATH if not set
            if [ -z "$GOPATH" ]; then
                export GOPATH="$HOME/go"
                log_info "GOPATH set to: $GOPATH"
            fi
            
            # Add GOPATH/bin to PATH if not present
            if [[ ":$PATH:" != *":$GOPATH/bin:"* ]]; then
                export PATH="$PATH:$GOPATH/bin"
                log_warning "Added $GOPATH/bin to PATH for this session"
                log_info "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
                echo "    export PATH=\$PATH:\$HOME/go/bin"
            fi
        else
            log_error "Go installation failed"
            return 1
        fi
    fi
}

# Check and install Docker
install_docker() {
    log_info "Checking Docker..."
    if command -v docker &> /dev/null; then
        log_success "Docker already installed: $(docker --version)"
        
        # Check if docker daemon is running
        if ! docker ps &> /dev/null; then
            log_warning "Docker daemon is not running"
            case "$OS" in
                linux)
                    log_info "Start Docker with: sudo systemctl start docker"
                    ;;
                macos)
                    log_info "Start Docker Desktop application"
                    ;;
            esac
        fi
    else
        log_info "Installing Docker..."
        case "$OS" in
            linux)
                # Docker installation varies by distro
                case "$PACKAGE_MANAGER" in
                    apt)
                        log_info "Installing Docker via convenience script..."
                        curl -fsSL https://get.docker.com -o /tmp/get-docker.sh
                        sudo sh /tmp/get-docker.sh
                        rm /tmp/get-docker.sh
                        sudo usermod -aG docker "$USER"
                        log_success "Docker installed. Log out and back in for group changes to take effect"
                        ;;
                    dnf|yum)
                        sudo $PACKAGE_MANAGER install -y docker
                        sudo systemctl start docker
                        sudo systemctl enable docker
                        sudo usermod -aG docker "$USER"
                        ;;
                    pacman)
                        sudo pacman -S --noconfirm docker
                        sudo systemctl start docker
                        sudo systemctl enable docker
                        sudo usermod -aG docker "$USER"
                        ;;
                    *)
                        log_warning "Please install Docker manually: https://docs.docker.com/engine/install/"
                        ;;
                esac
                ;;
            macos)
                log_info "Installing Docker Desktop..."
                brew install --cask docker
                log_info "Please start Docker Desktop application after installation"
                ;;
            *)
                log_warning "Please install Docker manually: https://docs.docker.com/engine/install/"
                ;;
        esac
    fi
}

# Check and install Docker Compose
install_docker_compose() {
    log_info "Checking Docker Compose..."
    
    # Check both docker-compose (standalone) and docker compose (plugin)
    if command -v docker-compose &> /dev/null; then
        log_success "Docker Compose already installed: $(docker-compose --version)"
    elif docker compose version &> /dev/null; then
        log_success "Docker Compose (plugin) already installed: $(docker compose version)"
    else
        log_info "Installing Docker Compose..."
        case "$OS" in
            linux)
                # Install Docker Compose plugin
                COMPOSE_VERSION=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
                sudo mkdir -p /usr/local/lib/docker/cli-plugins
                sudo curl -SL "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-linux-$(uname -m)" \
                    -o /usr/local/lib/docker/cli-plugins/docker-compose
                sudo chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
                log_success "Docker Compose installed: $(docker compose version)"
                ;;
            macos)
                # Docker Desktop includes Docker Compose
                log_success "Docker Compose included with Docker Desktop"
                ;;
            *)
                log_warning "Please install Docker Compose manually: https://docs.docker.com/compose/install/"
                ;;
        esac
    fi
}

# Install Go development tools
install_go_tools() {
    log_info "Installing Go development tools..."
    "$SCRIPT_DIR/install-tools.sh"
}

# Summary and next steps
show_summary() {
    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║  Installation Complete!                                        ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    
    log_success "Development environment is ready!"
    echo ""
    echo "Installed tools:"
    command -v git &> /dev/null && echo "  ✓ Git: $(git --version)"
    command -v make &> /dev/null && echo "  ✓ Make: $(make --version | head -1)"
    command -v go &> /dev/null && echo "  ✓ Go: $(go version)"
    command -v docker &> /dev/null && echo "  ✓ Docker: $(docker --version)"
    docker compose version &> /dev/null && echo "  ✓ Docker Compose: $(docker compose version)"
    command -v golangci-lint &> /dev/null && echo "  ✓ golangci-lint: installed"
    command -v goimports &> /dev/null && echo "  ✓ goimports: installed"
    command -v staticcheck &> /dev/null && echo "  ✓ staticcheck: installed"
    
    echo ""
    echo "Next steps:"
    echo "  1. Set up credentials:"
    echo "     The .pgpass file should already be configured"
    echo "     If not: ./scripts/update-pgpass.sh"
    echo ""
    echo "  2. Build and start services:"
    echo "     make docker-build"
    echo "     make docker-up"
    echo ""
    echo "  3. Verify services are running:"
    echo "     make docker-ps"
    echo ""
    echo "  4. View logs:"
    echo "     make docker-logs"
    echo ""
    echo "Documentation:"
    echo "  - Quick Start: QUICKSTART.md"
    echo "  - Architecture: docs/architecture/COORDINATOR_ENGINE_ARCHITECTURE.md"
    echo "  - Bootstrap Setup: docs/setup/BOOTSTRAP_SETUP.md"
    echo "  - Database Management: docs/setup/DATABASE_MANAGEMENT.md"
    echo ""
    
    if [ "$OS" = "linux" ] && groups | grep -q docker; then
        :
    elif [ "$OS" = "linux" ]; then
        log_warning "You may need to log out and back in for Docker group changes to take effect"
    fi
    
    if [[ ":$PATH:" != *":$HOME/go/bin:"* ]]; then
        log_warning "Remember to add Go binaries to your PATH permanently:"
        echo "    export PATH=\$PATH:\$HOME/go/bin"
    fi
}

# Main installation flow
main() {
    detect_os
    check_sudo
    
    echo "The following will be installed (if not present):"
    echo "  • Git"
    echo "  • Make"
    echo "  • Go 1.21+"
    echo "  • Docker"
    echo "  • Docker Compose"
    echo "  • Go development tools (linters, formatters, etc.)"
    echo ""
    read -p "Continue with installation? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Installation cancelled"
        exit 0
    fi
    
    echo ""
    install_package_manager || true
    install_git || log_error "Git installation failed, but continuing..."
    install_make || log_error "Make installation failed, but continuing..."
    install_go || log_error "Go installation failed, but continuing..."
    install_docker || log_warning "Docker installation incomplete"
    install_docker_compose || log_warning "Docker Compose installation incomplete"
    
    # Only install Go tools if Go is available
    if command -v go &> /dev/null; then
        install_go_tools || log_warning "Some Go tools may not have installed"
    fi
    
    show_summary
}

# Run main function
main
