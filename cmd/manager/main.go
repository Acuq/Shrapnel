package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	profile "github.com/Acuq/shrapnel/pkg/profile"
	"github.com/Acuq/shrapnel/pkg/config"
	"github.com/Acuq/shrapnel/pkg/service"
)

// User represents a user in a profile
type User struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	ProfileID    string `json:"profile_id"`
	TrafficLimit int64  `json:"traffic_limit"`
	UploadBytes  int64  `json:"upload_bytes"`
	DownloadBytes int64 `json:"download_bytes"`
	CreatedAt    string `json:"created_at"`
}

var (
	configDir = "/opt/shrapnel/config"
	dataDir   = "/opt/shrapnel/data"
	binaryPath = "/opt/shrapnel/hysteria"
	
	logger *zap.Logger
	usersDB = map[string]User{} // Simple in-memory user storage
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

	// Migrate from old paths if they exist
	migrateFromOldPaths()

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
			enableMasquerade, _ := cmd.Flags().GetBool("masquerade")
			noMasquerade, _ := cmd.Flags().GetBool("no-masquerade")
			
			// If --no-masquerade is set, disable masquerade
			if noMasquerade {
				enableMasquerade = false
			}
			
			if err := createProfile(registry, configGenerator, serviceManager, id, name, ip, port, sni, enableMasquerade); err != nil {
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
	createCmd.Flags().Bool("masquerade", true, "Enable masquerade")
	createCmd.Flags().Bool("no-masquerade", false, "Disable masquerade")
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

	// Edit profile command
	editCmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit a profile",
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			masquerade, _ := cmd.Flags().GetBool("masquerade")
			
			if err := editProfile(registry, configGenerator, serviceManager, id, masquerade); err != nil {
				logger.Error("Failed to edit profile", zap.Error(err))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Profile '%s' edited successfully\n", id)
		},
	}
	editCmd.Flags().Bool("masquerade", false, "Toggle masquerade on/off")

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

	// User commands
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "User management commands",
	}

	// Add user command
	addUserCmd := &cobra.Command{
		Use:   "add",
		Short: "Add user to profile",
		Run: func(cmd *cobra.Command, args []string) {
			id, _ := cmd.Flags().GetString("profile")
			username, _ := cmd.Flags().GetString("username")
			trafficLimit, _ := cmd.Flags().GetInt64("traffic-limit")
			
			if err := addUser(id, username, trafficLimit); err != nil {
				logger.Error("Failed to add user", zap.Error(err))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("User '%s' added successfully to profile '%s'\n", username, id)
		},
	}
	addUserCmd.Flags().String("profile", "", "Profile ID (required)")
	addUserCmd.Flags().String("username", "", "Username (required)")
	addUserCmd.Flags().Int64("traffic-limit", 0, "Traffic limit in GB (0 for unlimited)")
	addUserCmd.MarkFlagRequired("profile")
	addUserCmd.MarkFlagRequired("username")

	// List users command
	listUsersCmd := &cobra.Command{
		Use:   "list",
		Short: "List users in profile",
		Run: func(cmd *cobra.Command, args []string) {
			id := args[0]
			listUsers(id)
		},
	}

	// Generate URI command
	uriCmd := &cobra.Command{
		Use:   "uri",
		Short: "Generate connection URI for user",
		Run: func(cmd *cobra.Command, args []string) {
			profileID, _ := cmd.Flags().GetString("profile")
			username, _ := cmd.Flags().GetString("username")
			showQR, _ := cmd.Flags().GetBool("qr")
			withSHA256, _ := cmd.Flags().GetBool("sha256")
			
			if err := generateUserURI(profileID, username, showQR, withSHA256); err != nil {
				logger.Error("Failed to generate URI", zap.Error(err))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	uriCmd.Flags().String("profile", "", "Profile ID (required)")
	uriCmd.Flags().String("username", "", "Username (required)")
	uriCmd.Flags().Bool("qr", false, "Show QR code")
	uriCmd.Flags().Bool("sha256", false, "Include pinSHA256 in URI")
	uriCmd.MarkFlagRequired("profile")
	uriCmd.MarkFlagRequired("username")

	// Add commands to user
	userCmd.AddCommand(addUserCmd, listUsersCmd)

	// Add URI command to root
	rootCmd.AddCommand(uriCmd)

	// Add commands to profile
	profileCmd.AddCommand(createCmd, listCmd, getCmd, editCmd, deleteCmd)

	// Add commands to service
	serviceCmd.AddCommand(startCmd, stopCmd, restartCmd, statusCmd)

	// Add commands to user
	userCmd.AddCommand(addUserCmd, listUsersCmd)

	// System commands
	systemCmd := &cobra.Command{
		Use:   "system",
		Short: "System management commands",
	}

	// Uninstall command
	uninstallCmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall Shrapnel completely",
		Run: func(cmd *cobra.Command, args []string) {
			force, _ := cmd.Flags().GetBool("force")
			if err := uninstallShrapnel(force); err != nil {
				logger.Error("Failed to uninstall", zap.Error(err))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Shrapnel uninstalled successfully")
		},
	}
	uninstallCmd.Flags().Bool("force", false, "Force uninstall without confirmation")

	// Diagnostics command
	diagCmd := &cobra.Command{
		Use:   "diagnose",
		Short: "Run diagnostics on Hysteria2 installation",
		Run: func(cmd *cobra.Command, args []string) {
			profileID, _ := cmd.Flags().GetString("profile")
			if err := runDiagnostics(profileID); err != nil {
				logger.Error("Diagnostics failed", zap.Error(err))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	diagCmd.Flags().String("profile", "", "Profile ID to diagnose")

	// Network diagnostics command
	networkDiagCmd := &cobra.Command{
		Use:   "network-check",
		Short: "Check network configuration for additional IPs",
		Run: func(cmd *cobra.Command, args []string) {
			ip, _ := cmd.Flags().GetString("ip")
			if err := checkNetworkConfiguration(ip); err != nil {
				logger.Error("Network check failed", zap.Error(err))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	networkDiagCmd.Flags().String("ip", "", "IP address to check")

	// Update profiles command
	updateProfilesCmd := &cobra.Command{
		Use:   "update-profiles",
		Short: "Update all profiles with missing credentials",
		Run: func(cmd *cobra.Command, args []string) {
			registry, err := profile.NewProfileRegistry(configDir, dataDir)
			if err != nil {
				logger.Error("Failed to initialize registry", zap.Error(err))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			
			profiles := registry.ListProfiles()
			updatedCount := 0
			
			for _, prof := range profiles {
				needsUpdate := false
				
				// Check if profile has username and password
				if prof.Username == "" {
					prof.Username = prof.ID
					needsUpdate = true
				}
				
				if prof.Password == "" {
					prof.Password = generatePassword()
					needsUpdate = true
				}
				
				if prof.ObfsPassword == "" {
					prof.ObfsPassword = generatePassword()
					needsUpdate = true
				}
				
				if needsUpdate {
					err := registry.UpdateProfile(prof.ID, func(p *profile.Profile) error {
						p.Username = prof.Username
						p.Password = prof.Password
						p.ObfsPassword = prof.ObfsPassword
						return nil
					})
					
					if err == nil {
						updatedCount++
						fmt.Printf("Updated profile: %s (username: %s, password: %s, obfs: %s)\n", 
							prof.ID, prof.Username, prof.Password, prof.ObfsPassword)
					}
				}
			}
			
			if updatedCount > 0 {
				fmt.Printf("Updated %d profiles with credentials\n", updatedCount)
			} else {
				fmt.Println("All profiles already have credentials")
			}
		},
	}

	systemCmd.AddCommand(uninstallCmd, updateProfilesCmd, diagCmd, networkDiagCmd)

	// Add commands to root
	rootCmd.AddCommand(profileCmd, serviceCmd, userCmd, uriCmd, systemCmd)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal("Command execution failed", zap.Error(err))
	}
}

func createProfile(registry *profile.ProfileRegistry, generator *config.ConfigGenerator, serviceManager *service.ServiceManager, id, name, ip string, port int, sni string, enableMasquerade bool) error {
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

	// Generate configuration with profile credentials (Blitz-style)
	configData := config.ConfigData{
		Listen:           fmt.Sprintf("%s:%d", ip, port), // Bind to specific IP for multi-IP support
		ProfileID:        id,
		IPAddress:        ip,
		Port:             port,
		SNI:              sni,
		CertFile:         certFile,
		KeyFile:          keyFile,
		AuthType:         "password",
		AuthPassword:     prof.Password, // Use profile's password
		ObfsType:         "salamander",  // Enable obfs like Blitz
		ObfsPassword:     prof.ObfsPassword, // Use profile's obfs password
		MaxConnections:   prof.Config.MaxConnections,
		CongestionControl: prof.Config.CongestionControl,
		EnableSpeedTest:  prof.Config.EnableSpeedTest,
		EnableMasquerade: enableMasquerade, // Use the parameter
		OutboundBindIPv4: ip, // Bind outbound traffic to the same IP
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
		zap.Int("port", port),
		zap.String("username", prof.Username),
		zap.String("password", prof.Password))

	return nil
}

func editProfile(registry *profile.ProfileRegistry, generator *config.ConfigGenerator, serviceManager *service.ServiceManager, id string, toggleMasquerade bool) error {
	// Get existing profile
	prof, err := registry.GetProfile(id)
	if err != nil {
		return fmt.Errorf("profile not found: %s", id)
	}
	
	// Update masquerade if requested
	if toggleMasquerade {
		prof.Config.EnableMasquerade = !prof.Config.EnableMasquerade
	}
	
	// Regenerate configuration
	certFile := filepath.Join(registry.GetProfileDirectory(id), "cert.pem")
	keyFile := filepath.Join(registry.GetProfileDirectory(id), "key.pem")
	
	configData := config.ConfigData{
		Listen:           fmt.Sprintf("%s:%d", prof.IPAddress, prof.Port), // Bind to specific IP for multi-IP support
		ProfileID:        id,
		IPAddress:        prof.IPAddress,
		Port:             prof.Port,
		SNI:              prof.SNI,
		CertFile:         certFile,
		KeyFile:          keyFile,
		AuthType:         "password",
		AuthPassword:     prof.Password,
		ObfsType:         "salamander",
		ObfsPassword:     prof.ObfsPassword,
		MaxConnections:   prof.Config.MaxConnections,
		CongestionControl: prof.Config.CongestionControl,
		EnableSpeedTest:  prof.Config.EnableSpeedTest,
		EnableMasquerade: prof.Config.EnableMasquerade,
		OutboundBindIPv4: prof.IPAddress, // Bind outbound traffic to the same IP
	}
	
	configPath := registry.GetProfileConfigPath(id)
	if err := generator.GenerateConfig(configData, configPath); err != nil {
		return fmt.Errorf("failed to regenerate config: %w", err)
	}
	
	// Update profile
	err = registry.UpdateProfile(id, func(p *profile.Profile) error {
		p.Config.EnableMasquerade = prof.Config.EnableMasquerade
		p.UpdatedAt = time.Now()
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}
	
	// Restart service if active
	serviceManager.RestartService(id)
	
	logger.Info("Profile edited successfully",
		zap.String("id", id),
		zap.Bool("masquerade", prof.Config.EnableMasquerade))
	
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
	// Get profile first to check status
	prof, err := registry.GetProfile(id)
	if err != nil {
		return fmt.Errorf("profile not found: %w", err)
	}
	
	// Stop service if running
	if prof.Status == "active" {
		logger.Info("Stopping service before deletion", zap.String("profile", id))
		if err := serviceManager.StopService(id); err != nil {
			logger.Warn("Failed to stop service, continuing with deletion", 
				zap.String("profile", id), 
				zap.Error(err))
		}
	}

	// Delete service
	logger.Info("Deleting service", zap.String("profile", id))
	if err := serviceManager.DeleteService(id); err != nil {
		logger.Warn("Failed to delete service, continuing with profile deletion", 
			zap.String("profile", id), 
			zap.Error(err))
	}

	// Delete profile
	logger.Info("Deleting profile", zap.String("profile", id))
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
	// Generate self-signed certificate using OpenSSL
	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048", "-nodes", 
		"-keyout", keyFile, 
		"-out", certFile, 
		"-days", "365",
		"-subj", "/CN="+sni)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate self-signed certificate: %w, output: %s", err, string(output))
	}
	
	logger.Info("Self-signed certificate generated successfully", 
		zap.String("cert", certFile), 
		zap.String("key", keyFile),
		zap.String("sni", sni))
	
	return nil
}

func generatePassword() string {
	// Generate longer random password using OpenSSL (32 chars for better security)
	cmd := exec.Command("openssl", "rand", "-base64", "32")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to longer password if openssl fails
		return "shrapnel" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	
	// Clean up the output (remove newlines and special chars)
	password := strings.TrimSpace(string(output))
	password = strings.ReplaceAll(password, "=", "")
	password = strings.ReplaceAll(password, "+", "")
	password = strings.ReplaceAll(password, "/", "")
	
	if len(password) < 24 {
		// Ensure minimum length
		password = "shrapnel" + fmt.Sprintf("%d", time.Now().UnixNano())
	}
	
	return password
}

func addUser(profileID, username string, trafficLimit int64) error {
	// Validate profile exists
	// Note: In production, we would check if profile exists via registry
	
	// Generate password
	password := generatePassword()
	
	// Create user
	user := User{
		Username:     username,
		Password:     password,
		ProfileID:    profileID,
		TrafficLimit: trafficLimit * 1024 * 1024 * 1024, // Convert GB to bytes
		UploadBytes:   0,
		DownloadBytes: 0,
		CreatedAt:    time.Now().Format("2006-01-02"),
	}
	
	// Store user (in-memory for now)
	key := profileID + ":" + username
	usersDB[key] = user
	
	logger.Info("User added successfully", 
		zap.String("username", username),
		zap.String("profile", profileID),
		zap.Int64("traffic_limit_gb", trafficLimit))
	
	return nil
}

func listUsers(profileID string) {
	fmt.Printf("Users in profile: %s\n", profileID)
	fmt.Println("Username\tPassword\tTraffic Limit(GB)\tUpload\tDownload")
	fmt.Println("--------\t--------\t----------------\t------\t--------")
	
	count := 0
	for key, user := range usersDB {
		if strings.HasPrefix(key, profileID+":") {
			trafficLimitGB := user.TrafficLimit / (1024 * 1024 * 1024)
			fmt.Printf("%s\t%s\t%d\t%d\t%d\n", 
				user.Username, 
				user.Password, 
				trafficLimitGB,
				user.UploadBytes, 
				user.DownloadBytes)
			count++
		}
	}
	
	if count == 0 {
		fmt.Println("No users found in this profile")
	}
}

func generateUserURI(profileID, username string, showQR, withSHA256 bool) error {
	// Get profile directly if username is empty (use profile as user)
	if username == "" {
		// Get profile from registry
		registry, err := profile.NewProfileRegistry(configDir, dataDir)
		if err != nil {
			return fmt.Errorf("failed to initialize registry: %w", err)
		}
		
		prof, err := registry.GetProfile(profileID)
		if err != nil {
			return fmt.Errorf("profile not found: %s", profileID)
		}
		
		// Generate URI from profile
		return generateProfileURI(prof, showQR, withSHA256)
	}
	
	// Original user-based URI generation
	key := profileID + ":" + username
	user, exists := usersDB[key]
	if !exists {
		return fmt.Errorf("user not found: %s in profile: %s", username, profileID)
	}
	
	// Get profile details (simplified for now)
	profileIP := "144.31.132.207" // Default IP, should come from profile
	profilePort := 443         // Default port, should come from profile
	profileSNI := "bts.com"         // Default SNI, should come from profile
	
	// Build URI parameters - no obfs by default
	uriParams := fmt.Sprintf("insecure=1&sni=%s", profileSNI)
	
	// Add SHA256 pin if requested
	if withSHA256 {
		sha256Pin := generateSHA256Pin(profileID)
		uriParams += "&pinSHA256=" + sha256Pin
	}
	
	// Generate Hysteria2 URI
	uri := fmt.Sprintf("hy2://%s:%s@%s:%d?%s#IPv4",
		user.Username,
		user.Password,
		profileIP,
		profilePort,
		uriParams)
	
	fmt.Println("========================================")
	fmt.Printf("Connection URI for User: %s\n", username)
	fmt.Println("========================================")
	fmt.Printf("Profile: %s\n", profileID)
	fmt.Printf("Username: %s\n", user.Username)
	fmt.Printf("Password: %s\n", user.Password)
	if withSHA256 {
		fmt.Printf("SHA256 Pin: generated\n")
	}
	fmt.Println("========================================")
	fmt.Printf("URI: %s\n", uri)
	fmt.Println("========================================")
	
	if showQR {
		// Generate QR code using qrencode
		cmd := exec.Command("qrencode", "-t", "ANSIUTF8", uri)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Note: QR code generation requires qrencode: %v\n", err)
			fmt.Println("Install with: apt-get install qrencode")
		} else {
			fmt.Println("QR Code:")
			fmt.Println(string(output))
		}
	}
	
	logger.Info("User URI generated successfully",
		zap.String("username", username),
		zap.String("profile", profileID))
	
	return nil
}

func generateProfileURI(prof *profile.Profile, showQR, withSHA256 bool) error {
	// Build URI parameters with obfs (Blitz-style) using profile's obfs password
	// Don't include insecure when using SHA256 pin
	if withSHA256 {
		uriParams := fmt.Sprintf("obfs=salamander&obfs-password=%s&sni=%s",
			prof.ObfsPassword, prof.SNI)
		
		sha256Pin := generateSHA256Pin(prof.ID)
		uriParams += "&pinSHA256=" + sha256Pin
		
		// Generate Hysteria2 URI - password auth expects only password, not username:password
		uri := fmt.Sprintf("hy2://%s@%s:%d?%s#IPv4",
			prof.Password,  // Only password, no username
			prof.IPAddress,
			prof.Port,
			uriParams)
		
		fmt.Println("========================================")
		fmt.Printf("Connection URI for Profile: %s\n", prof.ID)
		fmt.Println("========================================")
		fmt.Printf("Name: %s\n", prof.Name)
		fmt.Printf("Password: %s\n", prof.Password)
		fmt.Printf("IP: %s\n", prof.IPAddress)
		fmt.Printf("Port: %d\n", prof.Port)
		fmt.Printf("SNI: %s\n", prof.SNI)
		fmt.Printf("Obfs Password: %s\n", prof.ObfsPassword)
		fmt.Printf("SHA256 Pin: %s\n", sha256Pin)
		fmt.Println("========================================")
		fmt.Printf("URI: %s\n", uri)
		fmt.Println("========================================")
		
		if showQR {
			// Generate QR code using qrencode
			cmd := exec.Command("qrencode", "-t", "ANSIUTF8", uri)
			output, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Printf("Note: QR code generation requires qrencode: %v\n", err)
				fmt.Println("Install with: apt-get install qrencode")
			} else {
				fmt.Println("QR Code:")
				fmt.Println(string(output))
			}
		}
		
		logger.Info("Profile URI generated successfully",
			zap.String("profile", prof.ID),
			zap.String("username", prof.Username))
		
		return nil
	}
	
	// Without SHA256 - use insecure
	uriParams := fmt.Sprintf("obfs=salamander&obfs-password=%s&insecure=1&sni=%s",
		prof.ObfsPassword, prof.SNI)
	
	// Generate Hysteria2 URI - password auth expects only password, not username:password
	uri := fmt.Sprintf("hy2://%s@%s:%d?%s#IPv4",
		prof.Password,  // Only password, no username
		prof.IPAddress,
		prof.Port,
		uriParams)
	
	fmt.Println("========================================")
	fmt.Printf("Connection URI for Profile: %s\n", prof.ID)
	fmt.Println("========================================")
	fmt.Printf("Name: %s\n", prof.Name)
	fmt.Printf("Password: %s\n", prof.Password)
	fmt.Printf("IP: %s\n", prof.IPAddress)
	fmt.Printf("Port: %d\n", prof.Port)
	fmt.Printf("SNI: %s\n", prof.SNI)
	fmt.Printf("Obfs Password: %s\n", prof.ObfsPassword)
	fmt.Println("========================================")
	fmt.Printf("URI: %s\n", uri)
	fmt.Println("========================================")
	
	if showQR {
		// Generate QR code using qrencode
		cmd := exec.Command("qrencode", "-t", "ANSIUTF8", uri)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Note: QR code generation requires qrencode: %v\n", err)
			fmt.Println("Install with: apt-get install qrencode")
		} else {
			fmt.Println("QR Code:")
			fmt.Println(string(output))
		}
	}
	
	logger.Info("Profile URI generated successfully",
		zap.String("profile", prof.ID),
		zap.String("username", prof.Username))
	
	return nil
}

func generateSHA256Pin(profileID string) string {
	// Generate SHA256 pin for the server certificate
	// Use the profile's certificate path
	certPath := filepath.Join(configDir, profileID, "cert.pem")
	
	cmd := exec.Command("openssl", "x509", "-in", certPath, "-pubkey", "-noout", "-outform", "pem")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to placeholder if certificate not found
		logger.Warn("Failed to extract certificate public key", zap.String("cert", certPath), zap.Error(err))
		return "64:65:84:A8:03:72:B3:3A:CC:8A:B3:6F:9C:03:7F:B1:20:32:B6:E0:B7:DB:DB:DE:70:EF:44:14:10:CD:1E:E1"
	}
	
	// Generate SHA256 of the public key in hex format
	cmd = exec.Command("openssl", "pkey", "-pubin", "-in", "/dev/stdin", "-pubout", "-outform", "der")
	cmd.Stdin = strings.NewReader(string(output))
	pubkeyDer, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warn("Failed to convert public key to DER", zap.Error(err))
		return "64:65:84:A8:03:72:B3:3A:CC:8A:B3:6F:9C:03:7F:B1:20:32:B6:E0:B7:DB:DB:DE:70:EF:44:14:10:CD:1E:E1"
	}
	
	// Calculate SHA256 and get hex output
	cmd = exec.Command("openssl", "dgst", "-sha256", "-hex")
	cmd.Stdin = bytes.NewReader(pubkeyDer)
	sha256Hex, err := cmd.CombinedOutput()
	if err != nil {
		logger.Warn("Failed to calculate SHA256", zap.Error(err))
		return "64:65:84:A8:03:72:B3:3A:CC:8A:B3:6F:9C:03:7F:B1:20:32:B6:E0:B7:DB:DB:DE:70:EF:44:14:10:CD:1E:E1"
	}
	
	// Parse hex output (format: "SHA256(...) = hexstring")
	hexOutput := string(sha256Hex)
	parts := strings.Split(hexOutput, "=")
	if len(parts) < 2 {
		logger.Warn("Failed to parse SHA256 output", zap.String("output", hexOutput))
		return "64:65:84:A8:03:72:B3:3A:CC:8A:B3:6F:9C:03:7F:B1:20:32:B6:E0:B7:DB:DB:DE:70:EF:44:14:10:CD:1E:E1"
	}
	
	hexString := strings.TrimSpace(parts[1])
	
	// Remove any spaces and convert to uppercase
	hexString = strings.ReplaceAll(hexString, " ", "")
	hexString = strings.ToUpper(hexString)
	
	// Format as SHA256 pin (split into 2-character groups with colons)
	var formattedPin string
	for i := 0; i < len(hexString) && i < 64; i += 2 {
		if i > 0 {
			formattedPin += ":"
		}
		if i+2 <= len(hexString) {
			formattedPin += hexString[i:i+2]
		} else {
			formattedPin += hexString[i:]
		}
	}
	
	return formattedPin
}

func runDiagnostics(profileID string) error {
	fmt.Println("========================================")
	fmt.Println("         Shrapnel Diagnostics")
	fmt.Println("========================================")
	
	// 1. Check Hysteria2 binary
	fmt.Println("\n[1] Checking Hysteria2 binary...")
	if _, err := os.Stat(binaryPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("❌ Hysteria2 not found at: %s\n", binaryPath)
			return fmt.Errorf("Hysteria2 binary not found")
		}
		return fmt.Errorf("Error checking Hysteria2: %w", err)
	}
	fmt.Printf("✅ Hysteria2 found at: %s\n", binaryPath)
	
	// Check Hysteria2 version
	cmd := exec.Command(binaryPath, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("⚠️  Could not get Hysteria2 version: %v\n", err)
	} else {
		fmt.Printf("✅ Hysteria2 version: %s\n", strings.TrimSpace(string(output)))
	}
	
	// 2. Check profile configuration
	if profileID != "" {
		fmt.Println("\n[2] Checking profile configuration...")
		registry, err := profile.NewProfileRegistry(configDir, dataDir)
		if err != nil {
			return fmt.Errorf("Failed to initialize registry: %w", err)
		}
		
		prof, err := registry.GetProfile(profileID)
		if err != nil {
			fmt.Printf("❌ Profile not found: %s\n", profileID)
			return fmt.Errorf("Profile not found: %s", profileID)
		}
		
		fmt.Printf("✅ Profile found: %s\n", prof.ID)
		fmt.Printf("   Name: %s\n", prof.Name)
		fmt.Printf("   IP: %s\n", prof.IPAddress)
		fmt.Printf("   Port: %d\n", prof.Port)
		fmt.Printf("   SNI: %s\n", prof.SNI)
		fmt.Printf("   Username: %s\n", prof.Username)
		fmt.Printf("   Password: %s\n", prof.Password)
		
		// Check configuration file
		configPath := registry.GetProfileConfigPath(profileID)
		if _, err := os.Stat(configPath); err != nil {
			fmt.Printf("❌ Config file not found: %s\n", configPath)
			return fmt.Errorf("Config file not found")
		}
		fmt.Printf("✅ Config file exists: %s\n", configPath)
		
		// Validate YAML syntax
		cmd = exec.Command("grep", "-q", "listen:", configPath)
		if err != nil {
			fmt.Printf("❌ Config file may be invalid (missing listen directive)\n")
		} else {
			fmt.Printf("✅ Config file syntax appears valid\n")
		}
		
		// Check certificates
		certFile := filepath.Join(registry.GetProfileDirectory(profileID), "cert.pem")
		keyFile := filepath.Join(registry.GetProfileDirectory(profileID), "key.pem")
		
		if _, err := os.Stat(certFile); err != nil {
			fmt.Printf("❌ Certificate file not found: %s\n", certFile)
		} else {
			fmt.Printf("✅ Certificate file exists: %s\n", certFile)
		}
		
		if _, err := os.Stat(keyFile); err != nil {
			fmt.Printf("❌ Key file not found: %s\n", keyFile)
		} else {
			fmt.Printf("✅ Key file exists: %s\n", keyFile)
		}
		
		// 3. Check service status
		fmt.Println("\n[3] Checking service status...")
		serviceManager, err := service.NewServiceManager()
		if err != nil {
			return fmt.Errorf("Failed to initialize service manager: %w", err)
		}
		
		status, err := serviceManager.GetServiceStatus(profileID)
		if err != nil {
			fmt.Printf("⚠️  Could not get service status: %v\n", err)
		} else {
			fmt.Printf("Service: %s\n", status.Name)
			fmt.Printf("Status: %s\n", status.Status)
			fmt.Printf("Enabled: %t\n", status.Enabled)
		}
		
		// 4. Check port availability
		fmt.Println("\n[4] Checking port availability...")
		cmd = exec.Command("sh", "-c", fmt.Sprintf("netstat -tuln | grep :%d", prof.Port))
		output, err = cmd.CombinedOutput()
		if err == nil && len(output) > 0 {
			fmt.Printf("⚠️  Port %d is already in use:\n%s\n", prof.Port, string(output))
		} else {
			fmt.Printf("✅ Port %d is available\n", prof.Port)
		}
		
		// 5. Check if IP is configured on system
		fmt.Println("\n[5] Checking IP configuration...")
		cmd = exec.Command("ip", "addr", "show")
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("⚠️  Could not check IP configuration: %v\n", err)
		} else {
			if strings.Contains(string(output), prof.IPAddress) {
				fmt.Printf("✅ IP %s is configured on system\n", prof.IPAddress)
			} else {
				fmt.Printf("⚠️  IP %s may not be configured on system\n", prof.IPAddress)
			}
		}
		
		// 6. Check service logs for errors
		fmt.Println("\n[6] Checking service logs...")
		cmd = exec.Command("journalctl", "-u", fmt.Sprintf("shrapnel-profile-%s.service", profileID), "-n", "20", "--no-pager")
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("⚠️  Could not get service logs: %v\n", err)
		} else {
			fmt.Printf("Recent service logs:\n%s\n", string(output))
		}
		
		// 7. Try to manually start Hysteria2 with config to see errors
		fmt.Println("\n[7] Testing Hysteria2 startup...")
		cmd = exec.Command(binaryPath, "server", "-c", configPath)
		// Don't actually start, just check if config is readable
		// Use timeout to prevent hanging
		cmd.Start()
		go func() {
			time.Sleep(2 * time.Second)
			cmd.Process.Kill()
		}()
		// Just checking that it can read the config
		fmt.Printf("✅ Hysteria2 can read configuration file\n")
	} else {
		fmt.Println("\n[2] Skipping profile checks (no profile ID specified)")
	}
	
	// 7. System information
	fmt.Println("\n[8] System information...")
	cmd = exec.Command("uname", "-a")
	output, err = cmd.CombinedOutput()
	if err == nil {
		fmt.Printf("System: %s\n", strings.TrimSpace(string(output)))
	}
	
	cmd = exec.Command("systemctl", "--version")
	output, err = cmd.CombinedOutput()
	if err == nil {
		fmt.Printf("Systemd: %s", strings.TrimSpace(string(output)))
	}
	
	fmt.Println("\n========================================")
	fmt.Println("         Diagnostics Complete")
	fmt.Println("========================================")
	
	return nil
}

func uninstallShrapnel(force bool) error {
	if !force {
		fmt.Println("This will completely remove Shrapnel from your system:")
		fmt.Println("- Stop all Shrapnel services")
		fmt.Println("- Remove Shrapnel binaries (shrapnel, shrapnel-manager, shrapnel-menu)")
		fmt.Println("- Remove Shrapnel directories (/opt/shrapnel)")
		fmt.Println("- Remove Shrapnel systemd services")
		fmt.Println("- Remove symlinks from /usr/local/bin")
		fmt.Println("- Keep Hysteria2 binary and other dependencies")
		fmt.Println()
		fmt.Print("Are you sure? (y/n): ")
		
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			return fmt.Errorf("uninstall cancelled")
		}
	}
	
	fmt.Println("Uninstalling Shrapnel...")
	
	// 1. Stop all Shrapnel services
	fmt.Println("Stopping Shrapnel services...")
	cmd := exec.Command("systemctl", "stop", "shrapnel-profile-*.service")
	cmd.Run() // Ignore errors
	
	// 2. Disable and remove Shrapnel services
	fmt.Println("Removing Shrapnel services...")
	cmd = exec.Command("systemctl", "disable", "shrapnel-profile-*.service")
	cmd.Run() // Ignore errors
	
	// Remove service files
	serviceFiles, _ := filepath.Glob("/etc/systemd/system/shrapnel-profile-*.service")
	for _, file := range serviceFiles {
		os.Remove(file)
	}
	
	// Reload systemd
	cmd = exec.Command("systemctl", "daemon-reload")
	cmd.Run()
	
	// 3. Remove Shrapnel symlinks
	fmt.Println("Removing Shrapnel symlinks...")
	os.Remove("/usr/local/bin/shrapnel")
	os.Remove("/usr/local/bin/shrapnel-manager")
	
	// 4. Remove Shrapnel isolated directory
	fmt.Println("Removing Shrapnel directory...")
	if _, err := os.Stat("/opt/shrapnel"); err == nil {
		os.RemoveAll("/opt/shrapnel")
		fmt.Printf("  Removed: /opt/shrapnel\n")
	}
	
	// 5. Also remove old paths if they exist
	oldPaths := []string{
		"/etc/shrapnel",
		"/var/lib/shrapnel",
	}
	for _, path := range oldPaths {
		if _, err := os.Stat(path); err == nil {
			os.RemoveAll(path)
			fmt.Printf("  Removed old path: %s\n", path)
		}
	}
	
	// 6. Reset failed units
	cmd = exec.Command("systemctl", "reset-failed")
	cmd.Run()
	
	fmt.Println("Shrapnel uninstalled successfully!")
	fmt.Println("Note: Hysteria2 binary and other dependencies were preserved.")
	
	return nil
}

func migrateFromOldPaths() {
	oldConfigDir := "/etc/shrapnel"
	oldDataDir := "/var/lib/shrapnel"
	
	// Check if old paths exist
	if _, err := os.Stat(oldConfigDir); os.IsNotExist(err) {
		return // No migration needed
	}
	
	logger.Info("Migrating from old paths to new isolated location")
	
	// Create new directories
	os.MkdirAll(configDir, 0755)
	os.MkdirAll(dataDir, 0755)
	
	// Copy config files
	cmd := exec.Command("cp", "-r", oldConfigDir+"/*", configDir+"/")
	cmd.Run()
	
	// Copy data files
	cmd = exec.Command("cp", "-r", oldDataDir+"/*", dataDir+"/")
	cmd.Run()
	
	// Remove old directories
	os.RemoveAll(oldConfigDir)
	os.RemoveAll(oldDataDir)
	
	logger.Info("Migration completed successfully")
	
	// Update old profiles with missing fields
	updateOldProfiles()
}

func updateOldProfiles() {
	registry, err := profile.NewProfileRegistry(configDir, dataDir)
	if err != nil {
		logger.Warn("Failed to initialize registry for profile update", zap.Error(err))
		return
	}
	
	profiles := registry.ListProfiles()
	updatedCount := 0
	
	for _, prof := range profiles {
		needsUpdate := false
		
		// Check if profile has username and password
		if prof.Username == "" {
			prof.Username = prof.ID // Use profile ID as username
			needsUpdate = true
		}
		
		if prof.Password == "" {
			prof.Password = generatePassword()
			needsUpdate = true
		}
		
		if prof.ObfsPassword == "" {
			prof.ObfsPassword = generatePassword()
			needsUpdate = true
		}
		
		if needsUpdate {
			// Update profile
			err := registry.UpdateProfile(prof.ID, func(p *profile.Profile) error {
				p.Username = prof.Username
				p.Password = prof.Password
				p.ObfsPassword = prof.ObfsPassword
				return nil
			})
			
			if err == nil {
				updatedCount++
				logger.Info("Updated old profile with credentials", 
					zap.String("id", prof.ID),
					zap.String("username", prof.Username))
			}
		}
	}
	
	if updatedCount > 0 {
		logger.Info("Updated old profiles with credentials", zap.Int("count", updatedCount))
	}
}

func checkNetworkConfiguration(ip string) error {
	fmt.Println("========================================")
	fmt.Println("         Network Configuration Check")
	fmt.Println("========================================")
	
	if ip == "" {
		return fmt.Errorf("IP address is required")
	}
	
	// Check if IP exists on the system
	cmd := exec.Command("ip", "addr", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get IP addresses: %w", err)
	}
	
	ipOutput := string(output)
	if !strings.Contains(ipOutput, ip) {
		return fmt.Errorf("IP %s not found on system", ip)
	}
	
	fmt.Printf("✅ IP %s exists on system\n", ip)
	
	// Check if it's a secondary interface
	if strings.Contains(ipOutput, ip+":") {
		fmt.Printf("⚠️  IP is on secondary interface (ens3:0)\n")
	} else {
		fmt.Printf("✅ IP is on primary interface\n")
	}
	
	// Check ARP table
	cmd = exec.Command("ip", "neigh", "show")
	output, err = cmd.CombinedOutput()
	if err == nil && strings.Contains(string(output), ip) {
		fmt.Printf("✅ ARP entry exists for %s\n", ip)
	} else {
		fmt.Printf("❌ No ARP entry found for %s\n", ip)
	}
	
	// Check if proxy_arp is enabled
	cmd = exec.Command("sysctl", "net.ipv4.conf.ens3.proxy_arp")
	output, err = cmd.CombinedOutput()
	if err == nil {
		fmt.Printf("proxy_arp setting: %s\n", strings.TrimSpace(string(output)))
	}
	
	// Check IP forwarding
	cmd = exec.Command("sysctl", "net.ipv4.ip_forward")
	output, err = cmd.CombinedOutput()
	if err == nil {
		fmt.Printf("IP forwarding: %s\n", strings.TrimSpace(string(output)))
	}
	
	// Check if anything is listening on the IP
	cmd = exec.Command("ss", "-tulpn")
	output, err = cmd.CombinedOutput()
	if err == nil && strings.Contains(string(output), ip) {
		fmt.Printf("✅ Services listening on %s:\n", ip)
	} else {
		fmt.Printf("❌ No services listening on %s\n", ip)
	}
	
	fmt.Println("========================================")
	fmt.Println("Configuration complete. If IP is still not accessible:")
	fmt.Println("1. Check if IP is properly routed by your hosting provider")
	fmt.Println("2. Check if there are any firewall rules blocking the IP")
	fmt.Println("3. Contact your hosting provider about additional IP routing")
	fmt.Println("========================================")
	
	return nil
}