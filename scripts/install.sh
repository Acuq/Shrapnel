#!/bin/bash

# Shrapnel Multi-IP Proxy Manager Installation Script
# Based on Blitz Panel installation approach

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/shrapnel"
DATA_DIR="/var/lib/shrapnel"
SERVICE_DIR="/etc/systemd/system"
SCRIPT_DIR="/usr/local/bin"

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}        🚀 Shrapnel Multi-IP Proxy Manager Installation 🚀${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}Please run as root${NC}"
    exit 1
fi

# Check OS
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
else
    echo -e "${RED}Cannot detect OS${NC}"
    exit 1
fi

echo -e "${YELLOW}Detected OS: $OS${NC}"

# Install dependencies
echo -e "${YELLOW}Installing dependencies...${NC}"
case $OS in
    ubuntu|debian)
        apt-get update
        apt-get install -y build-essential git wget curl systemd sqlite3
        ;;
    centos|rhel|fedora)
        yum install -y git wget curl systemd sqlite
        ;;
    arch|manjaro)
        pacman -S --noconfirm git wget curl systemd sqlite
        ;;
    *)
        echo -e "${RED}Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}Go not found. Installing Go 1.21...${NC}"
    wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
    rm -rf /usr/local/go
    tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
    rm go1.21.0.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
fi

# Create directories
echo -e "${YELLOW}Creating directories...${NC}"
mkdir -p "$CONFIG_DIR"
mkdir -p "$DATA_DIR"
mkdir -p "$SERVICE_DIR"
mkdir -p "$SCRIPT_DIR"

# Install Hysteria2 if not present
if ! command -v hysteria &> /dev/null; then
    echo -e "${YELLOW}Installing Hysteria2...${NC}"
    # Download latest hysteria2
    HYSTERIA_VERSION=$(curl -s https://api.github.com/repos/apernet/hysteria/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    wget "https://github.com/apernet/hysteria/releases/download/${HYSTERIA_VERSION}/hysteria-linux-amd64" -O "$INSTALL_DIR/hysteria"
    chmod +x "$INSTALL_DIR/hysteria"
    echo -e "${GREEN}✓ Hysteria2 installed${NC}"
else
    echo -e "${GREEN}✓ Hysteria2 already installed${NC}"
fi

# Build and install manager
echo -e "${YELLOW}Building Shrapnel Manager...${NC}"
# Assuming the script is run from the project directory
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_DIR"

# Build the manager
echo "Building manager..."
cd cmd/manager

# Add local package replacements and tidy
go mod edit -replace github.com/Acuq/shrapnel/pkg/profile="$PROJECT_DIR/pkg/profile"
go mod edit -replace github.com/Acuq/shrapnel/pkg/config="$PROJECT_DIR/pkg/config"
go mod edit -replace github.com/Acuq/shrapnel/pkg/service="$PROJECT_DIR/pkg/service"
go mod tidy

# Build the binary
go build -o "$INSTALL_DIR/shrapnel-manager" .
cd "$PROJECT_DIR"

chmod +x "$INSTALL_DIR/shrapnel-manager"
echo -e "${GREEN}✓ Manager installed${NC}"

# Install console panel
echo -e "${YELLOW}Installing console panel...${NC}"
cp "$PROJECT_DIR/scripts/menu.sh" "$SCRIPT_DIR/shrapnel-menu"
chmod +x "$SCRIPT_DIR/shrapnel-menu"
echo -e "${GREEN}✓ Console panel installed${NC}"

# Create symlink for easy access
ln -sf "$SCRIPT_DIR/shrapnel-menu" "$INSTALL_DIR/shrapnel"

# Set permissions
chown -R root:root "$CONFIG_DIR"
chown -R root:root "$DATA_DIR"
chmod 755 "$CONFIG_DIR"
chmod 755 "$DATA_DIR"

echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✓ Installation completed successfully!${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Usage:${NC}"
echo -e "  ${GREEN}shrapnel${NC}           - Launch console panel"
echo -e "  ${GREEN}shrapnel-manager${NC}   - Command line manager"
echo ""
echo -e "${YELLOW}Quick Start:${NC}"
echo -e "  1. Run ${GREEN}shrapnel${NC}"
echo -e "  2. Create a profile with IP address"
echo -e "  3. Start the profile service"
echo -e "  4. Add users to the profile"
echo ""
echo -e "${YELLOW}Configuration directories:${NC}"
echo -e "  Config: $CONFIG_DIR"
echo -e "  Data:   $DATA_DIR"
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"