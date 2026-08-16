package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	profile "github.com/Acuq/shrapnel/pkg/profile"
	"github.com/Acuq/shrapnel/pkg/config"
	"github.com/Acuq/shrapnel/pkg/service"
)

var (
	configDir = "/etc/shrapnel"
	dataDir   = "/var/lib/shrapnel"
	binaryPath = "/usr/local/bin/hysteria"
	
	logger *zap.Logger
)

func main() {
	// Initialize logger
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create directories
	if err := os.MkdirAll(configDir, 0755); err != nil {
		logger.Fatal("Failed to create config directory", zap.Error(err))
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Fatal("Failed to create data directory", zap.Error(err))
	}

	// Initialize components
	registry, err := profile.NewProfileRegistry(configDir, dataDir)
	if err != nil {
		logger.Fatal("Failed to initialize profile registry", zap.Error(err))
	}

	serviceManager, err := service.NewServiceManager()
	if err != nil {
		logger.Fatal("Failed to initialize service manager", zap.Error(err))
	}

	configGenerator, err := config.NewConfigGenerator()
	if err != nil {
		logger.Fatal("Failed to initialize config generator", zap.Error(err))
	}

	// Create root command
	rootCmd := &cobra.Command{
		Use:   "shrapnel-manager",
		Short: "Shrapnel Multi-IP Proxy Profile Manager",
		Long:  "Manage multiple isolated Hysteria2 profiles with different IP addresses",
	}

	// Profile commands
	profileCmd := &cobra.Command{
		Use:   "profile",
		Short: "Profile management commands",
	}

	// Create profile command
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new profile",
		Run: func(cmd *cobra.Command, args []string) {
			id, _ := cmd.Flags().GetString("id")
			name, _ := cmd.Flags().GetString("name")
			ip, _ := cmd.Flags().GetString("ip")
			port, _ := cmd.Flags().GetInt("port")
			sni, _ := cmd.Flags().GetString("sni")
			
			if err := createProfile(registry, configGenerator, serviceManager, id, name, ip, port, sni); err != nil {
				logger.Error("Failed to create profile", zap.Error(err))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			
			fmt.Printf("Profile '%s' created successfully\n", id)
		},
	}
	createCmd.Flags().String("id", "", "Profile ID (required)")
	createCmd.Flags().String("name", "", "Profile name (required)")
	createCmd.Flags().String("ip", "", "IP address (required)")
	createCmd.Flags().Int("port", 443, "Port number")
	createCmd.Flags().String("sni", "bts.com", "SNI")
	createCmd.MarkFlagRequired("id")
	createCmd.MarkFlagRequired("name")
	createCmd.MarkFlagRequired("ip")

	// List profiles command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		Run: func(cmd *cobra.Command, args []string) {
			listProfiles(registry)
		},
	}

	// Get profile command
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get profile details",
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			getProfile(registry, id)
		},
	}

	// Delete profile command
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a profile",
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			if err := deleteProfile(registry, serviceManager, id); err != nil {
				logger.Error("Failed to delete profile", zap.Error(err))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Profile '%s' deleted successfully\n", id)
		},
	}

	// Service commands
	serviceCmd := &cobra.Command{
		Use:   "service",
		Short: "Service management commands",
	}

	// Start service command
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start a profile service",
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			if err := startProfileService(registry, serviceManager, id); err != nil {
				logger.Error("Failed to start service", zap.Error(err))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Service for profile '%s' started successfully\n", id)
		},
	}

	// Stop service command
	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a profile service",
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			if err := stopProfileService(registry, serviceManager, id); err != nil {
				logger.Error("Failed to stop service", zap.Error(err))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Service for profile '%s' stopped successfully\n", id)
		},
	}

	// Restart service command
	restartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart a profile service",
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			if err := restartProfileService(registry, serviceManager, id); err != nil {
				logger.Error("Failed to restart service", zap.Error(err))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Service for profile '%s' restarted successfully\n", id)
		},
	}

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Get service status",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				id := args[0]
				getServiceStatus(serviceManager, id)
			} else {
				getAllServiceStatus(serviceManager)
			}
		},
	}

	// Add commands to profile
	profileCmd.AddCommand(createCmd, listCmd, getCmd, deleteCmd)

	// Add commands to service
	serviceCmd.AddCommand(startCmd, stopCmd, restartCmd, statusCmd)

	// Add to root
	rootCmd.AddCommand(profileCmd, serviceCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal("Command execution failed", zap.Error(err))
	}
}

func createProfile(registry *profile.ProfileRegistry, generator *config.ConfigGenerator, serviceManager *service.ServiceManager, id, name, ip string, port int, sni string) error {
	// Create profile
	prof, err := registry.CreateProfile(id, name, ip, port, sni)
	if err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	// Generate TLS certificates
	certFile := filepath.Join(registry.GetProfileDirectory(id), "cert.pem")
	keyFile := filepath.Join(registry.GetProfileDirectory(id), "key.pem")
	
	if err := generateSelfSignedCert(certFile, keyFile, sni); err != nil {
		return fmt.Errorf("failed to generate certificates: %w", err)
	}

	// Generate configuration
	configData := config.ConfigData{
		ProfileID:        id,
		IPAddress:        ip,
		Port:             port,
		SNI:              sni,
		CertFile:         certFile,
		KeyFile:          keyFile,
		AuthType:         "password",
		AuthPassword:     generatePassword(),
		MaxConnections:   prof.Config.MaxConnections,
		CongestionControl: prof.Config.CongestionControl,
		EnableSpeedTest:  prof.Config.EnableSpeedTest,
		EnableMasquerade: prof.Config.EnableMasquerade,
	}

	configPath := registry.GetProfileConfigPath(id)
	if err := generator.GenerateConfig(configData, configPath); err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	// Create systemd service
	if err := serviceManager.CreateService(id, configPath, binaryPath); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	logger.Info("Profile created successfully", 
		zap.String("id", id), 
		zap.String("ip", ip), 
		zap.Int("port", port))

	return nil
}

func listProfiles(registry *profile.ProfileRegistry) {
	profiles := registry.ListProfiles()
	
	if len(profiles) == 0 {
		fmt.Println("No profiles found")
		return
	}

	fmt.Println("Profiles:")
	fmt.Println("ID\tName\tIP\tPort\tStatus")
	fmt.Println("--\t----\t--\t----\t------")
	
	for _, prof := range profiles {
		fmt.Printf("%s\t%s\t%s\t%d\t%s\n", 
			prof.ID, prof.Name, prof.IPAddress, prof.Port, prof.Status)
	}
}

func getProfile(registry *profile.ProfileRegistry, id string) {
	prof, err := registry.GetProfile(id)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Profile Details:\n")
	fmt.Printf("ID: %s\n", prof.ID)
	fmt.Printf("Name: %s\n", prof.Name)
	fmt.Printf("IP Address: %s\n", prof.IPAddress)
	fmt.Printf("Port: %d\n", prof.Port)
	fmt.Printf("SNI: %s\n", prof.SNI)
	fmt.Printf("Status: %s\n", prof.Status)
	fmt.Printf("Created: %s\n", prof.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", prof.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Traffic Stats:\n")
	fmt.Printf("  Total: %d bytes\n", prof.TrafficStats.TotalBytes)
	fmt.Printf("  Upload: %d bytes\n", prof.TrafficStats.UploadBytes)
	fmt.Printf("  Download: %d bytes\n", prof.TrafficStats.DownloadBytes)
	fmt.Printf("  Active Connections: %d\n", prof.TrafficStats.ActiveConnections)
}

func deleteProfile(registry *profile.ProfileRegistry, serviceManager *service.ServiceManager, id string) error {
	// Delete service first
	if err := serviceManager.DeleteService(id); err != nil {
		logger.Warn("Failed to delete service", zap.String("profile", id), zap.Error(err))
	}

	// Delete profile
	if err := registry.DeleteProfile(id); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	logger.Info("Profile deleted successfully", zap.String("id", id))
	return nil
}

func startProfileService(registry *profile.ProfileRegistry, serviceManager *service.ServiceManager, id string) error {
	// Check if profile exists
	if _, err := registry.GetProfile(id); err != nil {
		return fmt.Errorf("profile not found: %w", err)
	}

	// Start service
	if err := serviceManager.StartService(id); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Update profile status
	if err := registry.SetProfileStatus(id, "active"); err != nil {
		logger.Warn("Failed to update profile status", zap.Error(err))
	}

	logger.Info("Service started successfully", zap.String("profile", id))
	return nil
}

func stopProfileService(registry *profile.ProfileRegistry, serviceManager *service.ServiceManager, id string) error {
	// Check if profile exists
	if _, err := registry.GetProfile(id); err != nil {
		return fmt.Errorf("profile not found: %w", err)
	}

	// Stop service
	if err := serviceManager.StopService(id); err != nil {
		return fmt.Errorf("failed to stop service: %w", err)
	}

	// Update profile status
	if err := registry.SetProfileStatus(id, "inactive"); err != nil {
		logger.Warn("Failed to update profile status", zap.Error(err))
	}

	logger.Info("Service stopped successfully", zap.String("profile", id))
	return nil
}

func restartProfileService(registry *profile.ProfileRegistry, serviceManager *service.ServiceManager, id string) error {
	// Check if profile exists
	if _, err := registry.GetProfile(id); err != nil {
		return fmt.Errorf("profile not found: %w", err)
	}

	// Restart service
	if err := serviceManager.RestartService(id); err != nil {
		return fmt.Errorf("failed to restart service: %w", err)
	}

	logger.Info("Service restarted successfully", zap.String("profile", id))
	return nil
}

func getServiceStatus(serviceManager *service.ServiceManager, id string) {
	status, err := serviceManager.GetServiceStatus(id)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Service Status:\n")
	fmt.Printf("Name: %s\n", status.Name)
	fmt.Printf("Status: %s\n", status.Status)
	fmt.Printf("Enabled: %t\n", status.Enabled)
}

func getAllServiceStatus(serviceManager *service.ServiceManager) {
	services, err := serviceManager.ListAllServices()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(services) == 0 {
		fmt.Println("No services found")
		return
	}

	fmt.Println("Services:")
	fmt.Println("Name\tStatus\tEnabled")
	fmt.Println("----\t------\t-------")
	
	for _, svc := range services {
		enabled := "no"
		if svc.Enabled {
			enabled = "yes"
		}
		fmt.Printf("%s\t%s\t%s\n", svc.Name, svc.Status, enabled)
	}
}

func generateSelfSignedCert(certFile, keyFile, sni string) error {
	// Placeholder for certificate generation
	// In production, use proper certificate generation
	return fmt.Errorf("certificate generation not implemented")
}

func generatePassword() string {
	// Placeholder for password generation
	// In production, use proper random password generation
	return "changeme123"
}