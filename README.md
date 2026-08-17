# Shrapnel 🚀

<img src="https://img.shields.io/badge/version-beta-yellow" alt="Version">
<img src="https://img.shields.io/badge/license-MIT-blue" alt="License">
<img src="https://img.shields.io/badge/go-1.21+-00ADD8?logo=go" alt="Go Version">

**Shrapnel** - advanced Hysteria2 fork with multi-IP address support for creating isolated access points.

## 🎯 What is Shrapnel?

Shrapnel is a system for managing multiple isolated Hysteria2 profiles on a single server. Each profile operates as an independent "shard" with its own IP address, users, and settings, providing complete isolation and management flexibility.

## ✨ Key Features

- **🔀 Multi-IP Support** - Create profiles on different IP addresses
- **🛡️ Complete Isolation** - Each profile has its own configuration, users, and statistics
- **🎛️ Console Panel** - User-friendly management interface in Blitz Panel style
- **⚙️ Service Management** - Automatic systemd service management

## 🚀 Quick Start

### Installation

```bash
git clone https://github.com/Acuq/shrapnel.git
cd shrapnel
sudo bash scripts/install.sh
```

### Usage

```bash
# Launch console panel
sudo shrapnel

# Or via CLI
sudo shrapnel-manager profile create --id client1 --name "Client 1" --ip 192.168.1.100 --port 443 --sni example.com
sudo shrapnel-manager service start client1
```

## 📖 Documentation

- [Full Documentation](README-MULTI-IP.md) - Comprehensive guide
- [Quick Start](QUICKSTART.md) - Step-by-step tutorial
- [Architecture](multi-ip-architecture.md) - Technical details
- [Implementation Status](IMPLEMENTATION_STATUS.md) - Current status

## 💡 Usage Examples

### Multi-tenant Provider

```bash
# Create a profile for each client
sudo shrapnel-manager profile create --id client1 --name "Client 1" --ip 192.168.1.100 --port 443 --sni client1.example.com
sudo shrapnel-manager profile create --id client2 --name "Client 2" --ip 192.168.1.101 --port 443 --sni client2.example.com

# Start services
sudo shrapnel-manager service start client1
sudo shrapnel-manager service start client2
```

### Load Balancing

```bash
# Create multiple profiles for load distribution
for i in {1..3}; do
    sudo shrapnel-manager profile create --id "lb$i" --name "Load Balancer $i" --ip "192.168.1.$((100+i))" --port 443 --sni lb.example.com
    sudo shrapnel-manager service start "lb$i"
done
```

## 🏗️ Architecture

```
shrapnel/
├── cmd/
│   └── manager/          # CLI manager
├── pkg/
│   ├── profile/          # Profile management
│   ├── config/           # Configuration generation
│   └── service/          # Service management
└── scripts/
    ├── menu.sh           # Console panel
    └── install.sh        # Installation script
```

## 🔧 Requirements

- Linux with systemd (Ubuntu, Debian, CentOS, RHEL, Fedora, Arch)
- Go 1.21+
- Root access
- Multiple IP addresses on the server

## 🛡️ Security

- Complete profile isolation at OS level
- systemd security settings (NoNewPrivileges, PrivateTmp)
- TLS certificate support for each profile
- Proper file permissions

## 🤝 Compatibility

- Fully compatible with original Hysteria2
- Supports all Hysteria2 features (Realm, Masquerade, Obfs, etc.)
- Can work alongside existing Hysteria2 installations

## 📄 License

MIT License (inherited from original Hysteria2)

## 🙏 Acknowledgments

- Original Hysteria2 project: https://github.com/apernet/hysteria
- Blitz Panel for console interface inspiration: https://github.com/ReturnFI/Blitz



---

**Shrapnel** - when one IP isn't enough, use multiple shards! 💥