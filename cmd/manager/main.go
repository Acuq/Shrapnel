package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	configDir = "/etc/shrapnel"
	dataDir   = "/var/lib/shrapnel"
	binaryPath = "/usr/local/bin/hysteria"
	
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
			
			if err := generateUserURI(profileID, username, showQR); err != nil {
				logger.Error("Failed to generate URI", zap.Error(err))
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}
	uriCmd.Flags().String("profile", "", "Profile ID (required)")
	uriCmd.Flags().String("username", "", "Username (required)")
	uriCmd.Flags().Bool("qr", false, "Show QR code")
	uriCmd.MarkFlagRequired("profile")
	uriCmd.MarkFlagRequired("username")

	// Add commands to user
	userCmd.AddCommand(addUserCmd, listUsersCmd)

	// Add URI command to root
	rootCmd.AddCommand(uriCmd)

	// Add commands to profile
	profileCmd.AddCommand(createCmd, listCmd, getCmd, deleteCmd)

	// Add commands to service
	serviceCmd.AddCommand(startCmd, stopCmd, restartCmd, statusCmd)

	// Add commands to user
	userCmd.AddCommand(addUserCmd, listUsersCmd)

	// Add commands to root
	rootCmd.AddCommand(profileCmd, serviceCmd, userCmd, uriCmd)

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
	// Stop service first
	logger.Info("Stopping service before deletion", zap.String("profile", id))
	if err := serviceManager.StopService(id); err != nil {
		logger.Warn("Failed to stop service, continuing with deletion", 
			zap.String("profile", id), 
			zap.Error(err))
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
	// Generate random password using OpenSSL
	cmd := exec.Command("openssl", "rand", "-base64", "12")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to simple password if openssl fails
		return "shrapnel123"
	}
	
	// Clean up the output (remove newlines and special chars)
	password := strings.TrimSpace(string(output))
	password = strings.ReplaceAll(password, "=", "")
	password = strings.ReplaceAll(password, "+", "")
	password = strings.ReplaceAll(password, "/", "")
	
	if len(password) < 8 {
		password = "shrapnel123"
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

func generateUserURI(profileID, username string, showQR bool) error {
	// Get user
	key := profileID + ":" + username
	user, exists := usersDB[key]
	if !exists {
		return fmt.Errorf("user not found: %s in profile: %s", username, profileID)
	}
	
	// Get profile details (simplified for now)
	// In production, we would get this from the profile registry
	profileIP := "144.31.132.207" // Default IP, should come from profile
	profilePort := 443         // Default port, should come from profile
	profileSNI := "bts.com"         // Default SNI, should come from profile
	
	// Generate Hysteria2 URI
	uri := fmt.Sprintf("hy2://%s:%s@%s:%d?obfs=salamander&obfs-password=changeme&insecure=1&sni=%s#IPv4",
		user.Username,
		user.Password,
		profileIP,
		profilePort,
		profileSNI)
	
	fmt.Println("========================================")
	fmt.Printf("Connection URI for User: %s\n", username)
	fmt.Println("========================================")
	fmt.Printf("Profile: %s\n", profileID)
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