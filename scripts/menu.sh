#!/bin/bash

# Shrapnel Multi-IP Proxy Manager Console Panel
# Based on Blitz Panel design for consistency

# Configuration
CONFIG_DIR="/opt/shrapnel/config"
DATA_DIR="/var/lib/shrapnel"
BINARY_PATH="/usr/local/bin/hysteria"
MANAGER_PATH="/usr/local/bin/shrapnel-manager"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# System info
OS=$(cat /etc/os-release | grep ^ID= | cut -d= -f2 | tr -d '"')
ARCH=$(uname -m)
CPU=$(lscpu | grep 'Model name' | cut -d: -f2 | xargs)
RAM=$(free -h | awk '/^Mem:/ {print $2}')
IP=$(hostname -I | awk '{print $1}')

# Check if manager is installed
check_manager() {
    if [ ! -f "$MANAGER_PATH" ]; then
        echo -e "${RED}Error: Shrapnel Manager not found at $MANAGER_PATH${NC}"
        echo "Please install the manager first"
        exit 1
    fi
}

# Display system information
display_system_info() {
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${PURPLE}         🚀 Shrapnel Multi-IP Proxy Manager 🚀${NC}"
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"

    printf "\033[0;32m• OS:  \033[0m%-20s \033[0;32m• ARCH:  \033[0m%-20s\n" "$OS" "$ARCH"
    printf "\033[0;32m• CPU: \033[0m%-20s \033[0;32m• RAM:   \033[0m%-20s\n" "$CPU" "$RAM"
    printf "\033[0;32m• IP:  \033[0m%-20s \033[0;32m• PROFILES:\033[0m%-20s\n" "$IP" "$(get_profile_count)"
    echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
}

# Get profile count
get_profile_count() {
    if [ -d "$CONFIG_DIR" ]; then
        count=$(find "$CONFIG_DIR" -mindepth 1 -maxdepth 1 -type d | wc -l)
        echo "$count"
    else
        echo "0"
    fi
}

# Main menu
main_menu() {
    while true; do
        clear
        display_system_info
        echo -e "${YELLOW}                   ☼ Main Menu ☼${NC}"
        echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}[1]${NC} Profile Management"
        echo -e "${GREEN}[2]${NC} Service Management"
        echo -e "${GREEN}[3]${NC} IP Management"
        echo -e "${RED}[6]${NC} Uninstall Shrapnel"
        echo -e "${RED}[0]${NC} Exit"
        echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
        echo -ne "${YELLOW}➜ Enter your option: ${NC}"
        
        read -r choice
        case $choice in
            1) profile_menu ;;
            2) service_menu ;;
            3) ip_menu ;;
            6) uninstall_shrapnel ;;
            0) exit 0 ;;
            *) echo -e "${RED}Invalid option. Please try again.${NC}" ;;
        esac
        echo -e "${YELLOW}Press Enter to continue...${NC}"
        read -r
    done
}

# Profile management menu
profile_menu() {
    while true; do
        clear
        display_system_info
        echo -e "${YELLOW}                   ☼ Profile Management ☼${NC}"
        echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}[1]${NC} Create Profile"
        echo -e "${GREEN}[2]${NC} List Profiles"
        echo -e "${GREEN}[3]${NC} View Profile Details"
        echo -e "${GREEN}[4]${NC} Edit Profile"
        echo -e "${GREEN}[5]${NC} Show Profile URI"
        echo -e "${GREEN}[6]${NC} Update All Profiles"
        echo -e "${GREEN}[7]${NC} Delete Profile"
        echo -e "${RED}[0]${NC} Back to Main Menu"
        echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
        echo -ne "${YELLOW}➜ Enter your option: ${NC}"
        
        read -r choice
        case $choice in
            1) create_profile ;;
            2) list_profiles ;;
            3) view_profile ;;
            4) edit_profile ;;
            5) show_profile_uri ;;
            6) update_all_profiles ;;
            7) delete_profile ;;
            0) break ;;
            *) echo -e "${RED}Invalid option. Please try again.${NC}" ;;
        esac
        if [ "$choice" != "0" ]; then
            echo -e "${YELLOW}Press Enter to continue...${NC}"
            read -r
        fi
    done
}

# Create profile
create_profile() {
    echo -e "${CYAN}Create New Profile${NC}"
    echo -e "${CYAN}─────────────────────────────────────────────────────────────────${NC}"
    
    # Profile Name (used internally as the profile ID too)
    while true; do
        read -p "Enter Profile Name (alphanumeric, no spaces): " profile_id
        if [[ -z "$profile_id" ]]; then
            echo -e "${RED}Profile Name cannot be empty.${NC}"
        elif [[ "$profile_id" =~ ^[a-zA-Z0-9_-]+$ ]]; then
            if $MANAGER_PATH profile get "$profile_id" 2>&1 | grep -q "^ID:"; then
                echo -e "${RED}A profile named '${profile_id}' already exists. Choose another name.${NC}"
            else
                break
            fi
        else
            echo -e "${RED}Invalid name. Use only letters, numbers, hyphens, and underscores.${NC}"
        fi
    done
    profile_name="$profile_id"
    
    # Check for IPv6 availability
    has_ipv6=$(ip -6 addr show | grep -oP '(?<=inet6\s)[0-9a-fA-F:]+' | grep -v '^::1' | grep -v '^fe80' | head -1)
    has_ipv4=$(ip -4 addr show | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | head -1)
    
    # IP Type Selection
    ip_type="ipv4"
    if [ -n "$has_ipv6" ] && [ -n "$has_ipv4" ]; then
        echo ""
        echo "Available IP types:"
        echo "1) IPv4"
        echo "2) IPv6"
        while true; do
            read -p "Select IP type (1-2, default: 1): " ip_choice
            ip_choice=${ip_choice:-1}
            case $ip_choice in
                1) ip_type="ipv4"; break ;;
                2) ip_type="ipv6"; break ;;
                *) echo -e "${RED}Invalid choice. Enter 1 or 2.${NC}" ;;
            esac
        done
    elif [ -n "$has_ipv6" ]; then
        echo -e "${YELLOW}Note: Only IPv6 is available on this system${NC}"
        ip_type="ipv6"
    else
        echo -e "${YELLOW}Note: Only IPv4 is available on this system${NC}"
        ip_type="ipv4"
    fi
    
    # IP Address input based on type
    if [ "$ip_type" == "ipv4" ]; then
        while true; do
            read -p "Enter IPv4 Address: " ip_address
            if [[ "$ip_address" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]]; then
                if [[ $(echo "$ip_address" | awk -F. '{for (i=1;i<=NF;i++) if ($i>255) exit 1}') ]]; then
                    echo -e "${RED}Invalid IPv4 address. Values must be between 0 and 255.${NC}"
                else
                    break
                fi
            else
                echo -e "${RED}Invalid IPv4 address format.${NC}"
            fi
        done
    else
        while true; do
            read -p "Enter IPv6 Address: " ip_address
            # More comprehensive IPv6 validation
            # Check for valid characters and basic structure
            if [[ "$ip_address" =~ ^[0-9a-fA-F:]+$ ]] && [[ "$ip_address" == *:* ]] && [[ "$ip_address" != ":::" ]]; then
                # Remove any subnet mask if present
                clean_ip="${ip_address%%/*}"
                
                # Count colons (should be between 2 and 7 for valid IPv6)
                colon_count=$(echo "$clean_ip" | tr -cd ':' | wc -c)
                
                # Allow :: compression which reduces colon count
                if [[ "$clean_ip" == *"::"* ]]; then
                    # With :: compression, minimum 2 colons, maximum 7
                    if [ "$colon_count" -ge 2 ] && [ "$colon_count" -le 7 ]; then
                        break
                    else
                        echo -e "${RED}Invalid IPv6 address format. Check colon count.${NC}"
                    fi
                else
                    # Without :: compression, should be exactly 7 colons (8 groups)
                    if [ "$colon_count" -eq 7 ]; then
                        break
                    else
                        echo -e "${RED}Invalid IPv6 address format. Expected 7 colons for full notation.${NC}"
                    fi
                fi
            else
                echo -e "${RED}Invalid IPv6 address format. Use proper IPv6 notation (e.g., 2001:db8::1).${NC}"
            fi
        done
    fi
    
    # Port
    while true; do
        read -p "Enter Port (default: 443): " port
        port=${port:-443}
        if [[ "$port" =~ ^[0-9]+$ ]] && [ "$port" -ge 1 ] && [ "$port" -le 65535 ]; then
            break
        else
            echo -e "${RED}Invalid port number. Enter a number between 1 and 65535.${NC}"
        fi
    done
    
    # SNI
    read -p "Enter SNI (default: bts.com): " sni
    sni=${sni:-bts.com}
    
    # Masquerade
    read -p "Enable Masquerade? (Y/n, default: Y): " enable_masquerade
    enable_masquerade=${enable_masquerade:-Y}
    
    if [[ "$enable_masquerade" =~ ^[Yy]$ ]]; then
        masquerade_flag="--masquerade"
    else
        masquerade_flag="--no-masquerade"
    fi
    
    # Create profile with appropriate IP flag
    echo -e "${YELLOW}Creating profile with $ip_type address...${NC}"
    if [ "$ip_type" == "ipv6" ]; then
        if $MANAGER_PATH profile create --id "$profile_id" --name "$profile_name" --ipv6 "$ip_address" --port "$port" --sni "$sni" $masquerade_flag; then
            echo -e "${GREEN}✓ Profile created successfully!${NC}"
            
            # Ask if user wants to start the service
            read -p "Start the service now? (y/n): " start_service
            if [[ "$start_service" =~ ^[Yy]$ ]]; then
                if $MANAGER_PATH service start "$profile_id"; then
                    echo -e "${GREEN}✓ Service started successfully!${NC}"
                else
                    echo -e "${RED}✗ Failed to start service${NC}"
                fi
            fi
        else
            echo -e "${RED}✗ Failed to create profile${NC}"
        fi
    else
        if $MANAGER_PATH profile create --id "$profile_id" --name "$profile_name" --ip "$ip_address" --port "$port" --sni "$sni" $masquerade_flag; then
            echo -e "${GREEN}✓ Profile created successfully!${NC}"
            
            # Ask if user wants to start the service
            read -p "Start the service now? (y/n): " start_service
            if [[ "$start_service" =~ ^[Yy]$ ]]; then
                if $MANAGER_PATH service start "$profile_id"; then
                    echo -e "${GREEN}✓ Service started successfully!${NC}"
                else
                    echo -e "${RED}✗ Failed to start service${NC}"
                fi
            fi
        else
            echo -e "${RED}✗ Failed to create profile${NC}"
        fi
    fi
}

# List profiles
list_profiles() {
    echo -e "${CYAN}Existing Profiles${NC}"
    echo -e "${CYAN}─────────────────────────────────────────────────────────────────${NC}"
    
    $MANAGER_PATH profile list
}

# View profile details
view_profile() {
    read -p "Enter Profile Name: " profile_id
    
    if [ -z "$profile_id" ]; then
        echo -e "${RED}Profile Name cannot be empty${NC}"
        return
    fi
    
    $MANAGER_PATH profile get "$profile_id"
}

# Edit profile
edit_profile() {
    read -p "Enter Profile Name to edit: " profile_id
    
    if [ -z "$profile_id" ]; then
        echo -e "${RED}Profile Name cannot be empty${NC}"
        return
    fi
    
    echo -e "${YELLOW}Edit Profile${NC}"
    echo -e "${CYAN}─────────────────────────────────────────────────────────────────${NC}"
    echo "Available options:"
    echo "1. Change Port"
    echo "2. Change SNI"
    echo "3. Enable/Disable Masquerade"
    echo "4. Enable/Disable Speed Test"
    echo "0. Cancel"
    
    read -p "Select option: " edit_choice
    
    case $edit_choice in
        1) change_profile_port "$profile_id" ;;
        2) change_profile_sni "$profile_id" ;;
        3) toggle_masquerade "$profile_id" ;;
        4) toggle_speedtest "$profile_id" ;;
        0) echo "Cancelled" ;;
        *) echo -e "${RED}Invalid option${NC}" ;;
    esac
}

# Delete profile
delete_profile() {
    read -p "Enter Profile Name to delete: " profile_id
    
    if [ -z "$profile_id" ]; then
        echo -e "${RED}Profile Name cannot be empty${NC}"
        return
    fi
    
    # Confirm deletion
    read -p "Are you sure you want to delete profile '$profile_id'? (y/n): " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        echo "Deletion cancelled"
        return
    fi
    
    # Stop service first if running
    echo -e "${YELLOW}Stopping service if running...${NC}"
    if $MANAGER_PATH service stop "$profile_id" 2>/dev/null; then
        echo -e "${GREEN}✓ Service stopped${NC}"
    else
        echo -e "${YELLOW}Service was not running or failed to stop${NC}"
    fi
    
    # Delete profile
    if $MANAGER_PATH profile delete "$profile_id"; then
        echo -e "${GREEN}✓ Profile deleted successfully${NC}"
    else
        echo -e "${RED}✗ Failed to delete profile${NC}"
    fi
}

# Update all profiles
update_all_profiles() {
    echo -e "${YELLOW}Updating all profiles with missing credentials...${NC}"
    $MANAGER_PATH system update-profiles
}

# Show profile URI
show_profile_uri() {
    read -p "Enter Profile Name: " profile_id
    
    if [ -z "$profile_id" ]; then
        echo -e "${RED}Profile Name cannot be empty${NC}"
        return
    fi
    
    read -p "Show QR code? (y/n): " show_qr
    qr_flag=""
    if [[ "$show_qr" =~ ^[Yy]$ ]]; then
        qr_flag="--qr"
    fi
    
    read -p "Include SHA256 pin? (y/n, default: n): " show_sha256
    sha256_flag=""
    if [[ "$show_sha256" =~ ^[Yy]$ ]]; then
        sha256_flag="--sha256"
    fi
    
    echo -e "${YELLOW}Generating URI for profile '$profile_id'...${NC}"
    $MANAGER_PATH uri --profile "$profile_id" --username "" $qr_flag $sha256_flag
}

# Service management menu
service_menu() {
    while true; do
        clear
        display_system_info
        echo -e "${YELLOW}                   ☼ Service Management ☼${NC}"
        echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}[1]${NC} Start Service"
        echo -e "${GREEN}[2]${NC} Stop Service"
        echo -e "${GREEN}[3]${NC} Restart Service"
        echo -e "${GREEN}[4]${NC} View Service Status"
        echo -e "${GREEN}[5]${NC} View Service Logs"
        echo -e "${GREEN}[6]${NC} View All Services Status"
        echo -e "${RED}[0]${NC} Back to Main Menu"
        echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
        echo -ne "${YELLOW}➜ Enter your option: ${NC}"
        
        read -r choice
        case $choice in
            1) start_service ;;
            2) stop_service ;;
            3) restart_service ;;
            4) view_service_status ;;
            5) view_service_logs ;;
            6) view_all_services ;;
            0) break ;;
            *) echo -e "${RED}Invalid option. Please try again.${NC}" ;;
        esac
        if [ "$choice" != "0" ]; then
            echo -e "${YELLOW}Press Enter to continue...${NC}"
            read -r
        fi
    done
}

# Start service
start_service() {
    read -p "Enter Profile Name: " profile_id
    
    if [ -z "$profile_id" ]; then
        echo -e "${RED}Profile Name cannot be empty${NC}"
        return
    fi
    
    if $MANAGER_PATH service start "$profile_id"; then
        echo -e "${GREEN}✓ Service started successfully${NC}"
    else
        echo -e "${RED}✗ Failed to start service${NC}"
    fi
}

# Stop service
stop_service() {
    read -p "Enter Profile Name: " profile_id
    
    if [ -z "$profile_id" ]; then
        echo -e "${RED}Profile Name cannot be empty${NC}"
        return
    fi
    
    if $MANAGER_PATH service stop "$profile_id"; then
        echo -e "${GREEN}✓ Service stopped successfully${NC}"
    else
        echo -e "${RED}✗ Failed to stop service${NC}"
    fi
}

# Restart service
restart_service() {
    read -p "Enter Profile Name: " profile_id
    
    if [ -z "$profile_id" ]; then
        echo -e "${RED}Profile Name cannot be empty${NC}"
        return
    fi
    
    if $MANAGER_PATH service restart "$profile_id"; then
        echo -e "${GREEN}✓ Service restarted successfully${NC}"
    else
        echo -e "${RED}✗ Failed to restart service${NC}"
    fi
}

# View service status
view_service_status() {
    read -p "Enter Profile Name: " profile_id
    
    if [ -z "$profile_id" ]; then
        echo -e "${RED}Profile Name cannot be empty${NC}"
        return
    fi
    
    $MANAGER_PATH service status "$profile_id"
}

# View service logs
view_service_logs() {
    read -p "Enter Profile Name: " profile_id
    
    if [ -z "$profile_id" ]; then
        echo -e "${RED}Profile Name cannot be empty${NC}"
        return
    fi
    
    read -p "Number of lines (default: 50): " lines
    lines=${lines:-50}
    
    service_name="shrapnel-profile-${profile_id}.service"
    journalctl -u "$service_name" -n "$lines" --no-pager
}

# View all services
view_all_services() {
    $MANAGER_PATH service status
}

# IP management menu
ip_menu() {
    while true; do
        clear
        display_system_info
        echo -e "${YELLOW}                   ☼ IP Management ☼${NC}"
        echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
        echo -e "${GREEN}[1]${NC} List Available IPs"
        echo -e "${GREEN}[2]${NC} Check IP Availability"
        echo -e "${GREEN}[3]${NC} Assign IP to Profile"
        echo -e "${RED}[0]${NC} Back to Main Menu"
        echo -e "${CYAN}═══════════════════════════════════════════════════════════════${NC}"
        echo -ne "${YELLOW}➜ Enter your option: ${NC}"
        
        read -r choice
        case $choice in
            1) list_available_ips ;;
            2) check_ip_availability ;;
            3) assign_ip_to_profile ;;
            0) break ;;
            *) echo -e "${RED}Invalid option. Please try again.${NC}" ;;
        esac
        if [ "$choice" != "0" ]; then
            echo -e "${YELLOW}Press Enter to continue...${NC}"
            read -r
        fi
    done
}

# List available IPs
list_available_ips() {
    echo -e "${CYAN}Available IP Addresses${NC}"
    echo -e "${CYAN}─────────────────────────────────────────────────────────────────${NC}"
    
    # IPv4 Addresses
    echo -e "${GREEN}IPv4 Addresses:${NC}"
    ipv4_addresses=$(ip -4 addr show | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | sort -u)
    if [ -n "$ipv4_addresses" ]; then
        echo "$ipv4_addresses"
    else
        echo -e "${YELLOW}No IPv4 addresses found${NC}"
    fi
    
    echo ""
    
    # IPv6 Addresses
    echo -e "${GREEN}IPv6 Addresses:${NC}"
    ipv6_addresses=$(ip -6 addr show | grep -oP '(?<=inet6\s)[0-9a-fA-F:]+(|/[0-9]+)' | grep -v '^::1' | grep -v '^fe80' | sort -u)
    if [ -n "$ipv6_addresses" ]; then
        echo "$ipv6_addresses"
    else
        echo -e "${YELLOW}No IPv6 addresses found${NC}"
    fi
    
    echo ""
    echo -e "${YELLOW}Note: This shows all configured IPs on the system${NC}"
}

# Check IP availability
check_ip_availability() {
    read -p "Enter IP Address to check: " ip_address
    
    if [ -z "$ip_address" ]; then
        echo -e "${RED}IP address cannot be empty${NC}"
        return
    fi
    
    # Determine IP type
    if [[ "$ip_address" =~ : ]]; then
        ip_type="IPv6"
        ip_check_cmd="ip -6 addr show"
    else
        ip_type="IPv4"
        ip_check_cmd="ip -4 addr show"
    fi
    
    echo -e "${CYAN}Checking $ip_type address: $ip_address${NC}"
    
    # Check if IP is configured on system
    if $ip_check_cmd | grep -q "$ip_address"; then
        echo -e "${GREEN}✓ IP is configured on system${NC}"
    else
        echo -e "${RED}✗ IP is not configured on system${NC}"
    fi
    
    # Check if IP is used by any profile
    if [ -d "$CONFIG_DIR" ]; then
        # Check both IPv4 and IPv6 fields in profile configs
        used_by_ipv4=$(grep -r "ip_address: \"$ip_address\"" "$CONFIG_DIR" 2>/dev/null | cut -d: -f1 | xargs -I{} basename {})
        used_by_ipv6=$(grep -r "ipv6_address: \"$ip_address\"" "$CONFIG_DIR" 2>/dev/null | cut -d: -f1 | xargs -I{} basename {})
        
        used_by=""
        [ -n "$used_by_ipv4" ] && used_by="$used_by_ipv4"
        [ -n "$used_by_ipv6" ] && used_by="$used_by $used_by_ipv6"
        
        if [ -n "$used_by" ]; then
            echo -e "${YELLOW}⚠ IP is used by profile(s): $used_by${NC}"
        else
            echo -e "${GREEN}✓ IP is not used by any profile${NC}"
        fi
    fi
}

# Assign IP to profile
assign_ip_to_profile() {
    read -p "Enter Profile Name: " profile_id
    read -p "Enter new IP Address: " ip_address
    
    if [ -z "$profile_id" ] || [ -z "$ip_address" ]; then
        echo -e "${RED}Profile Name and IP address cannot be empty${NC}"
        return
    fi
    
    echo -e "${YELLOW}This feature requires profile reconfiguration${NC}"
    echo "For now, please delete and recreate the profile with the new IP"
}

# Helper functions for profile editing
change_profile_port() {
    local profile_id=$1
    read -p "Enter new port: " new_port
    
    if [[ "$new_port" =~ ^[0-9]+$ ]] && [ "$new_port" -ge 1 ] && [ "$new_port" -le 65535 ]; then
        echo -e "${YELLOW}Changing port to $new_port${NC}"
        echo "Implementation pending - requires profile update functionality"
    else
        echo -e "${RED}Invalid port number${NC}"
    fi
}

change_profile_sni() {
    local profile_id=$1
    read -p "Enter new SNI: " new_sni
    
    if [ -n "$new_sni" ]; then
        echo -e "${YELLOW}Changing SNI to $new_sni${NC}"
        echo "Implementation pending - requires profile update functionality"
    else
        echo -e "${RED}SNI cannot be empty${NC}"
    fi
}

toggle_masquerade() {
    local profile_id=$1
    echo -e "${YELLOW}Toggle masquerade for profile: $profile_id${NC}"
    $MANAGER_PATH profile edit "$profile_id" --masquerade
}

toggle_speedtest() {
    local profile_id=$1
    echo -e "${YELLOW}Toggle speed test for profile: $profile_id${NC}"
    echo "Implementation pending - requires profile update functionality"
}

# Uninstall Shrapnel
uninstall_shrapnel() {
    echo -e "${RED}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${RED}            ⚠️  UNINSTALL SHRAPNEL ⚠️${NC}"
    echo -e "${RED}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "${YELLOW}This will completely remove Shrapnel from your system:${NC}"
    echo -e "${YELLOW}• Stop all Shrapnel services${NC}"
    echo -e "${YELLOW}• Remove Shrapnel binaries${NC}"
    echo -e "${YELLOW}• Remove Shrapnel directories (/opt/shrapnel)${NC}"
    echo -e "${YELLOW}• Remove Shrapnel systemd services${NC}"
    echo -e "${YELLOW}• Remove symlinks from /usr/local/bin${NC}"
    echo -e "${GREEN}• Keep Hysteria2 binary and other dependencies${NC}"
    echo ""
    echo -e "${YELLOW}All profiles and data will be deleted!${NC}"
    echo ""
    read -p "Are you sure you want to uninstall Shrapnel? (yes/no): " confirm
    
    if [ "$confirm" != "yes" ]; then
        echo -e "${GREEN}Uninstall cancelled${NC}"
        return
    fi
    
    echo -e "${YELLOW}Uninstalling Shrapnel...${NC}"
    
    # Use the CLI uninstall command
    if $MANAGER_PATH system uninstall --force; then
        echo -e "${GREEN}✓ Shrapnel uninstalled successfully${NC}"
        echo ""
        echo -e "${YELLOW}To reinstall, run:${NC}"
        echo -e "${GREEN}cd /tmp && git clone https://github.com/Acuq/Shrapnel.git && cd Shrapnel && sudo bash scripts/install.sh${NC}"
    else
        echo -e "${RED}✗ Uninstall failed${NC}"
    fi
}

# Main execution
check_manager
main_menu