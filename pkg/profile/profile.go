package profile

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Profile represents an isolated Hysteria2 instance
type Profile struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	IPAddress   string    `json:"ip_address"`
	IPv6Address string    `json:"ipv6_address,omitempty"`
	Port        int       `json:"port"`
	SNI         string    `json:"sni"`
	Status      string    `json:"status"` // active, inactive, error
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	
	// User credentials for direct connection
	Username    string `json:"username"`
	Password    string `json:"password"`
	ObfsPassword string `json:"obfs_password"` // Added for obfs support
	
	// Traffic statistics
	TrafficStats TrafficStats `json:"traffic_stats"`
	
	// Configuration
	Config ProfileConfig `json:"config"`
}

// ProfileConfig contains profile-specific configuration
type ProfileConfig struct {
	ObfsType          string `json:"obfs_type,omitempty"`
	ObfsPassword      string `json:"obfs_password,omitempty"`
	MaxConnections    int    `json:"max_connections,omitempty"`
	EnableMasquerade  bool   `json:"enable_masquerade,omitempty"`
	EnableSpeedTest   bool   `json:"enable_speed_test,omitempty"`
	CongestionControl string `json:"congestion_control,omitempty"`
}

// TrafficStats contains traffic statistics for a profile
type TrafficStats struct {
	TotalBytes       int64 `json:"total_bytes"`
	UploadBytes      int64 `json:"upload_bytes"`
	DownloadBytes    int64 `json:"download_bytes"`
	ActiveConnections int  `json:"active_connections"`
}

// ProfileRegistry manages all profiles
type ProfileRegistry struct {
	mu       sync.RWMutex
	profiles map[string]*Profile
	configDir string
	dataDir   string
}

// NewProfileRegistry creates a new profile registry
func NewProfileRegistry(configDir, dataDir string) (*ProfileRegistry, error) {
	registry := &ProfileRegistry{
		profiles:  make(map[string]*Profile),
		configDir: configDir,
		dataDir:   dataDir,
	}
	
	// Create directories if they don't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}
	
	// Load existing profiles
	if err := registry.loadProfiles(); err != nil {
		return nil, fmt.Errorf("failed to load profiles: %w", err)
	}
	
	return registry, nil
}

// CreateProfile creates a new profile
func (r *ProfileRegistry) CreateProfile(id, name, ipAddress string, useIPv6 bool, port int, sni string) (*Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if profile already exists
	if _, exists := r.profiles[id]; exists {
		return nil, fmt.Errorf("profile with ID %s already exists", id)
	}
	
	// Validate IP address
	if ipAddress == "" {
		return nil, fmt.Errorf("IP address cannot be empty")
	}
	
	// Validate port
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port number: %d", port)
	}
	
	// Generate username and password for the profile
	username := id
	password := generateRandomPassword()
	obfsPassword := generateRandomPassword()
	
	// Create profile
	profile := &Profile{
		ID:          id,
		Name:        name,
		Port:        port,
		SNI:         sni,
		Username:    username,
		Password:    password,
		ObfsPassword: obfsPassword,
		Status:      "inactive",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		TrafficStats: TrafficStats{},
		Config: ProfileConfig{
			MaxConnections:    1000,
			EnableSpeedTest:   true,
			CongestionControl: "bbr",
		},
	}
	
	// Set IP address based on type
	if useIPv6 {
		profile.IPv6Address = ipAddress
		profile.IPAddress = "" // Set IPv4 to empty for IPv6-only
	} else {
		profile.IPAddress = ipAddress
		profile.IPv6Address = "" // Clear IPv6 for IPv4-only profiles
	}
	
	// Create profile directory
	profileDir := filepath.Join(r.configDir, id)
	if err := os.MkdirAll(profileDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create profile directory: %w", err)
	}
	
	// Save profile metadata
	if err := r.saveProfileMetadata(profile); err != nil {
		return nil, fmt.Errorf("failed to save profile metadata: %w", err)
	}
	
	// Add to registry
	r.profiles[id] = profile
	
	// Save registry
	if err := r.saveRegistry(); err != nil {
		delete(r.profiles, id)
		return nil, fmt.Errorf("failed to save registry: %w", err)
	}
	
	return profile, nil
}

// GetProfile retrieves a profile by ID
func (r *ProfileRegistry) GetProfile(id string) (*Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	profile, exists := r.profiles[id]
	if !exists {
		return nil, fmt.Errorf("profile not found: %s", id)
	}
	
	return profile, nil
}

// ListProfiles returns all profiles
func (r *ProfileRegistry) ListProfiles() []*Profile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	profiles := make([]*Profile, 0, len(r.profiles))
	for _, profile := range r.profiles {
		profiles = append(profiles, profile)
	}
	
	return profiles
}

// UpdateProfile updates an existing profile
func (r *ProfileRegistry) UpdateProfile(id string, updates func(*Profile) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	profile, exists := r.profiles[id]
	if !exists {
		return fmt.Errorf("profile not found: %s", id)
	}
	
	// Apply updates
	if err := updates(profile); err != nil {
		return err
	}
	
	profile.UpdatedAt = time.Now()
	
	// Save profile metadata
	if err := r.saveProfileMetadata(profile); err != nil {
		return fmt.Errorf("failed to save profile metadata: %w", err)
	}
	
	// Save registry
	if err := r.saveRegistry(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}
	
	return nil
}

// DeleteProfile removes a profile
func (r *ProfileRegistry) DeleteProfile(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	_, exists := r.profiles[id]
	if !exists {
		return fmt.Errorf("profile not found: %s", id)
	}
	
	// Remove profile directory
	profileDir := filepath.Join(r.configDir, id)
	if err := os.RemoveAll(profileDir); err != nil {
		return fmt.Errorf("failed to remove profile directory: %w", err)
	}
	
	// Remove from registry
	delete(r.profiles, id)
	
	// Save registry
	if err := r.saveRegistry(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}
	
	return nil
}

// SetProfileStatus updates the status of a profile
func (r *ProfileRegistry) SetProfileStatus(id, status string) error {
	return r.UpdateProfile(id, func(p *Profile) error {
		p.Status = status
		return nil
	})
}

// UpdateTrafficStats updates traffic statistics for a profile
func (r *ProfileRegistry) UpdateTrafficStats(id string, uploadBytes, downloadBytes int64, activeConnections int) error {
	return r.UpdateProfile(id, func(p *Profile) error {
		p.TrafficStats.UploadBytes += uploadBytes
		p.TrafficStats.DownloadBytes += downloadBytes
		p.TrafficStats.TotalBytes += uploadBytes + downloadBytes
		p.TrafficStats.ActiveConnections = activeConnections
		return nil
	})
}

// saveProfileMetadata saves profile metadata to disk
func (r *ProfileRegistry) saveProfileMetadata(profile *Profile) error {
	profileDir := filepath.Join(r.configDir, profile.ID)
	metadataPath := filepath.Join(profileDir, "profile.json")
	
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profile metadata: %w", err)
	}
	
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write profile metadata: %w", err)
	}
	
	return nil
}

// loadProfiles loads all profiles from disk
func (r *ProfileRegistry) loadProfiles() error {
	registryPath := filepath.Join(r.configDir, "profiles.json")
	
	data, err := os.ReadFile(registryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing profiles
		}
		return fmt.Errorf("failed to read registry: %w", err)
	}
	
	var profileIDs []string
	if err := json.Unmarshal(data, &profileIDs); err != nil {
		return fmt.Errorf("failed to unmarshal registry: %w", err)
	}
	
	for _, id := range profileIDs {
		profileDir := filepath.Join(r.configDir, id)
		metadataPath := filepath.Join(profileDir, "profile.json")
		
		profileData, err := os.ReadFile(metadataPath)
		if err != nil {
			fmt.Printf("Warning: failed to read profile %s: %v\n", id, err)
			continue
		}
		
		var profile Profile
		if err := json.Unmarshal(profileData, &profile); err != nil {
			fmt.Printf("Warning: failed to unmarshal profile %s: %v\n", id, err)
			continue
		}
		
		r.profiles[id] = &profile
	}
	
	return nil
}

// saveRegistry saves the profile registry to disk
func (r *ProfileRegistry) saveRegistry() error {
	registryPath := filepath.Join(r.configDir, "profiles.json")
	
	profileIDs := make([]string, 0, len(r.profiles))
	for id := range r.profiles {
		profileIDs = append(profileIDs, id)
	}
	
	data, err := json.MarshalIndent(profileIDs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registry: %w", err)
	}
	
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry: %w", err)
	}
	
	return nil
}

// GetProfileDirectory returns the directory path for a profile
func (r *ProfileRegistry) GetProfileDirectory(id string) string {
	return filepath.Join(r.configDir, id)
}

// GetProfileConfigPath returns the config file path for a profile
func (r *ProfileRegistry) GetProfileConfigPath(id string) string {
	return filepath.Join(r.GetProfileDirectory(id), "config.yaml")
}

// GetProfileUsersDBPath returns the users database path for a profile
func (r *ProfileRegistry) GetProfileUsersDBPath(id string) string {
	return filepath.Join(r.dataDir, fmt.Sprintf("%s_users.db", id))
}

// generateRandomPassword generates a random password
func generateRandomPassword() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback to time-based password if crypto rand fails
		return fmt.Sprintf("shrapnel%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)[:32]
}