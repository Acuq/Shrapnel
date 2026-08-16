# Shrapnel Multi-IP Proxy Architecture

## Overview
System for multi-IP address support for Hysteria2 with isolated access point profiles.

## Core Components

### 1. Profile System
Each profile represents an isolated Hysteria2 access point:

**Profile Structure:**
```
/etc/shrapnel/
├── profile1/
│   ├── config.yaml          # Hysteria2 configuration
│   ├── users.db             # User database
│   ├── profile.json         # Profile metadata
│   └── shrapnel-profile-profile1.service # Systemd service
├── profile2/
│   ├── config.yaml
│   ├── users.db
│   ├── profile.json
│   └── shrapnel-profile-profile2.service
└── profiles.json            # Profile registry
```

**Profile Metadata (profile.json):**
```json
{
  "id": "profile1",
  "name": "Main Profile",
  "ip_address": "192.168.1.100",
  "port": 443,
  "sni": "example.com",
  "status": "active",
  "created_at": "2024-01-01T00:00:00Z",
  "traffic_stats": {
    "total_bytes": 0,
    "active_connections": 0
  }
}
```

### 2. Multi-IP Configuration
**Server Configuration:**
```yaml
listen: "[IP_ADDRESS]:443"
outbounds:
  - name: direct
    type: direct
    direct:
      bindIPv4: "[IP_ADDRESS]"
      mode: auto
```

### 3. Console Panel
Interface management based on Bash (Blitz Panel style):

**Main Menu:**
```
=== Shrapnel Multi-IP Manager ===
1. Profile Management
2. User Management  
3. Service Management
4. IP Management
5. Traffic Monitoring
0. Exit
```

**Profile Management:**
- Create new profile
- List all profiles
- Edit profile settings
- Delete profile
- Start/Stop profile service

**User Management:**
- Add user to specific profile
- Edit user in profile
- Remove user from profile
- List users in profile
- Reset user traffic

### 4. Service Architecture
**Multi-Instance Support:**
- Each profile runs as separate systemd service
- Service naming: `shrapnel-profile-[ID].service`
- Automatic port management to avoid conflicts
- Proper isolation and security settings

**Port Allocation:**
- Automatic port assignment
- Port availability checking
- Default port range: 10000-20000

### 5. IP Binding
**IP Assignment:**
- Static IP binding to profiles
- IP availability verification
- Automatic network configuration

**Network Configuration:**
```go
// Outbound configuration with IP binding
serverConfigOutboundDirect{
    Mode: "auto",
    BindIPv4: profile.IPAddress,
    BindIPv6: profile.IPv6Address,
}
```

## Implementation Plan

### Phase 1: Core Profile System ✅
1. Profile management library (Go)
2. Profile storage and retrieval
3. Profile validation
4. Basic CRUD operations

### Phase 2: Multi-IP Configuration ✅
1. Dynamic config generation
2. IP binding implementation
3. Port management
4. Network validation

### Phase 3: Console Panel ✅
1. Main menu system
2. Profile management UI
3. User management UI
4. Service control UI

### Phase 4: Service Integration ✅
1. Systemd service generation
2. Service lifecycle management
3. Log management
4. Status monitoring

## Technical Details

### File Structure
```
shrapnel/
├── cmd/
│   ├── server/              # Modified hysteria2 server
│   └── manager/             # Profile manager CLI
├── pkg/
│   ├── profile/             # Profile management
│   ├── config/              # Config generation
│   └── service/             # Service management
├── scripts/
│   ├── menu.sh              # Console panel
│   └── install.sh           # Installation script
└── configs/
    └── template.yaml        # Config template
```

### Key Features
1. **Isolation**: Complete user and traffic isolation between profiles
2. **Scalability**: Support for 100+ profiles on single server
3. **Management**: Unified console panel for all profiles
4. **Flexibility**: Different outbound types for different profiles
5. **Monitoring**: Separate traffic statistics per profile

### Dependencies
- Go 1.21+
- systemd (for Linux)
- SQLite (for user storage)
- Bash 4.0+ (for console panel)

## Compatibility
- Compatible with original Hysteria2
- Supports all Hysteria2 features (Realm, Masquerade, etc.)
- Can work parallel with existing Hysteria2 installations