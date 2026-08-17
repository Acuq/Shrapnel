# Shrapnel 🚀

<img src="https://img.shields.io/badge/version-beta-yellow" alt="Version">
<img src="https://img.shields.io/badge/license-MIT-blue" alt="License">
<img src="https://img.shields.io/badge/go-1.21+-00ADD8?logo=go" alt="Go Version">

**Shrapnel** - advanced Hysteria2 fork with multi-IP address support for creating isolated access points.

## 🎯 What is Shrapnel?

Shrapnel lets you run multiple independent Hysteria2 servers on a single VPS, each bound to its own dedicated IP address. Instead of juggling manual configs, systemd units, and certificates by hand, you manage everything — creating, editing, starting/stopping, and connecting to profiles — through one CLI and an interactive console menu.

Each profile is a fully separate Hysteria2 instance: its own IP, port, TLS certificate, obfuscation password, and auth credentials — so traffic for one profile never mixes with another, and you can run as many as you have IPs for.

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
```

## 📖 Documentation


- [Architecture](multi-ip-architecture.md) - Technical details


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