package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ServiceManager manages systemd services for Hysteria2 profiles
type ServiceManager struct {
	systemdPath string
	serviceDir   string
}

// NewServiceManager creates a new service manager
func NewServiceManager() (*ServiceManager, error) {
	// Check if systemd is available
	if _, err := exec.LookPath("systemctl"); err != nil {
		return nil, fmt.Errorf("systemd not found: %w", err)
	}
	
	return &ServiceManager{
		systemdPath: "/usr/bin/systemctl",
		serviceDir:  "/etc/systemd/system",
	}, nil
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Name        string
	Status      string // active, inactive, failed, unknown
	Enabled     bool
	Description string
}

// CreateService creates a systemd service for a profile
func (m *ServiceManager) CreateService(profileID, configPath, binaryPath string) error {
	serviceName := m.getServiceName(profileID)
	serviceFile := filepath.Join(m.serviceDir, serviceName)
	
	// Generate service file content
	serviceContent, err := generateServiceContent(profileID, configPath, binaryPath)
	if err != nil {
		return fmt.Errorf("failed to generate service content: %w", err)
	}
	
	// Write service file
	if err := os.WriteFile(serviceFile, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}
	
	// Reload systemd daemon
	if err := m.reloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
	
	return nil
}

// StartService starts a profile service
func (m *ServiceManager) StartService(profileID string) error {
	serviceName := m.getServiceName(profileID)
	
	// Enable service first
	if err := m.EnableService(profileID); err != nil {
		// Log warning but continue
	}
	
	cmd := exec.Command(m.systemdPath, "start", serviceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start service: %w, output: %s", err, string(output))
	}
	
	// Wait a moment for service to start
	time.Sleep(2 * time.Second)
	
	return nil
}

// StopService stops a profile service
func (m *ServiceManager) StopService(profileID string) error {
	serviceName := m.getServiceName(profileID)
	
	// Check if service exists and is loaded
	status, err := m.GetServiceStatus(profileID)
	if err != nil || status.Status == "unknown" {
		// Service doesn't exist or isn't loaded, consider it as stopped
		return nil
	}
	
	// Only try to stop if it's active
	if status.Status != "active" {
		return nil
	}
	
	cmd := exec.Command(m.systemdPath, "stop", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If error is "Unit not loaded", service is already stopped
		if strings.Contains(string(output), "Unit not loaded") {
			return nil
		}
		return fmt.Errorf("failed to stop service: %w, output: %s", err, string(output))
	}
	
	return nil
}

// RestartService restarts a profile service
func (m *ServiceManager) RestartService(profileID string) error {
	serviceName := m.getServiceName(profileID)
	
	cmd := exec.Command(m.systemdPath, "restart", serviceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restart service: %w, output: %s", err, string(output))
	}
	
	// Wait a moment for service to restart
	time.Sleep(2 * time.Second)
	
	return nil
}

// EnableService enables a profile service to start on boot
func (m *ServiceManager) EnableService(profileID string) error {
	serviceName := m.getServiceName(profileID)
	
	cmd := exec.Command(m.systemdPath, "enable", serviceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable service: %w, output: %s", err, string(output))
	}
	
	return nil
}

// DisableService disables a profile service from starting on boot
func (m *ServiceManager) DisableService(profileID string) error {
	serviceName := m.getServiceName(profileID)
	
	cmd := exec.Command(m.systemdPath, "disable", serviceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to disable service: %w, output: %s", err, string(output))
	}
	
	return nil
}

// DeleteService removes a profile service
func (m *ServiceManager) DeleteService(profileID string) error {
	serviceName := m.getServiceName(profileID)
	serviceFile := filepath.Join(m.serviceDir, serviceName)
	
	// Stop service if running
	status, err := m.GetServiceStatus(profileID)
	if err == nil && status.Status == "active" {
		if err := m.StopService(profileID); err != nil {
			return fmt.Errorf("failed to stop service before deletion: %w", err)
		}
	}
	
	// Disable service
	_ = m.DisableService(profileID)
	
	// Remove service file
	if err := os.Remove(serviceFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service file: %w", err)
	}
	
	// Reload systemd daemon
	if err := m.reloadDaemon(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}
	
	return nil
}

// GetServiceStatus returns the status of a profile service
func (m *ServiceManager) GetServiceStatus(profileID string) (*ServiceStatus, error) {
	serviceName := m.getServiceName(profileID)
	
	// Check if service exists
	serviceFile := filepath.Join(m.serviceDir, serviceName)
	if _, err := os.Stat(serviceFile); os.IsNotExist(err) {
		return &ServiceStatus{
			Name:   serviceName,
			Status: "unknown",
		}, nil
	}
	
	// Get service status
	cmd := exec.Command(m.systemdPath, "is-active", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ServiceStatus{
			Name:   serviceName,
			Status: "unknown",
		}, nil
	}
	
	status := strings.TrimSpace(string(output))
	
	// Check if service is enabled
	cmd = exec.Command(m.systemdPath, "is-enabled", serviceName)
	enabledOutput, err := cmd.CombinedOutput()
	enabled := err == nil && strings.TrimSpace(string(enabledOutput)) == "enabled"
	
	return &ServiceStatus{
		Name:    serviceName,
		Status:  status,
		Enabled: enabled,
	}, nil
}

// ListAllServices returns status of all Hysteria2 profile services
func (m *ServiceManager) ListAllServices() ([]*ServiceStatus, error) {
	cmd := exec.Command(m.systemdPath, "list-units", "--all", "--type=service", "shrapnel-profile-*.service")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	
	var services []*ServiceStatus
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		if strings.Contains(line, "shrapnel-profile-") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				serviceName := parts[0]
				status := parts[2]
				
				services = append(services, &ServiceStatus{
					Name:   serviceName,
					Status: status,
				})
			}
		}
	}
	
	return services, nil
}

// GetServiceLogs returns the logs for a profile service
func (m *ServiceManager) GetServiceLogs(profileID string, lines int) (string, error) {
	serviceName := m.getServiceName(profileID)
	
	cmd := exec.Command("journalctl", "-u", serviceName, "-n", fmt.Sprintf("%d", lines), "--no-pager")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get service logs: %w", err)
	}
	
	return string(output), nil
}

// reloadDaemon reloads the systemd daemon
func (m *ServiceManager) reloadDaemon() error {
	cmd := exec.Command(m.systemdPath, "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to reload daemon: %w, output: %s", err, string(output))
	}
	return nil
}

// getServiceName returns the systemd service name for a profile
func (m *ServiceManager) getServiceName(profileID string) string {
	return fmt.Sprintf("shrapnel-profile-%s.service", profileID)
}

// generateServiceContent generates the content for a systemd service file
func generateServiceContent(profileID, configPath, binaryPath string) (string, error) {
	return fmt.Sprintf(`[Unit]
Description=Shrapnel Multi-IP Profile: %s
After=network.target

[Service]
Type=simple
User=root
ExecStart=%s server -c %s
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536

# Security settings
NoNewPrivileges=true
PrivateTmp=true

# Environment
Environment="SHRAPNEL_PROFILE_ID=%s"

[Install]
WantedBy=multi-user.target
`, profileID, binaryPath, configPath, profileID), nil
}

// SetupARPProxy sets up ARP proxy for additional IP addresses
func (m *ServiceManager) SetupARPProxy(ipAddress string) error {
	// Check if IP is an additional IP (secondary interface)
	cmd := exec.Command("ip", "addr", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get IP addresses: %w", err)
	}
	
	ipOutput := string(output)
	
	// Check if this IP is on a secondary interface (contains colon)
	if !strings.Contains(ipOutput, ipAddress+":") {
		// This is the primary IP, no ARP proxy needed
		return nil
	}
	
	// Find the interface name
	lines := strings.Split(ipOutput, "\n")
	var interfaceName string
	for _, line := range lines {
		if strings.Contains(line, ipAddress) {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				interfaceName = strings.TrimSuffix(parts[1], ":")
				break
			}
		}
	}
	
	if interfaceName == "" {
		return fmt.Errorf("could not find interface for IP %s", ipAddress)
	}
	
	// Remove existing ARP entry if present
	cmd = exec.Command("ip", "neigh", "del", ipAddress, "dev", interfaceName)
	cmd.Run() // Ignore errors
	
	// Add ARP proxy entry
	cmd = exec.Command("ip", "neigh", "add", "proxy", ipAddress, "dev", interfaceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add ARP proxy: %w", err)
	}
	
	// Enable proxy_arp on the interface
	cmd = exec.Command("sysctl", "-w", fmt.Sprintf("net.ipv4.conf.%s.proxy_arp=1", interfaceName))
	if err := cmd.Run(); err != nil {
		// Log warning but don't fail
		fmt.Printf("Warning: failed to enable proxy_arp on %s: %v\n", interfaceName, err)
	}
	
	// Enable IP forwarding
	cmd = exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: failed to enable IP forwarding: %v\n", err)
	}
	
	return nil
}