# Quick Start

## One-Line Install

```bash
curl -sSL https://raw.githubusercontent.com/tudorAbrudan/tracelog/main/scripts/install.sh | bash
```

This will:
1. Download the latest TraceLog binary
2. Create a system user and data directory
3. Auto-detect Docker, web servers, and common log files
4. Create an admin account
5. Set up and start a systemd service
6. Open the firewall port if needed

## Manual Install

### Download

```bash
# Option 1: Go install
go install github.com/tudorAbrudan/tracelog/cmd/tracelog@latest

# Option 2: Download from releases (tar.gz from GoReleaser)
curl -sL -o tracelog.tgz "https://github.com/tudorAbrudan/tracelog/releases/latest/download/tracelog_linux_amd64.tar.gz"
tar -xzf tracelog.tgz tracelog
chmod +x tracelog
sudo mv tracelog /usr/local/bin/
```

### Upgrade in place

```bash
sudo tracelog upgrade
```

Downloads the latest GitHub release for your OS/architecture, verifies `checksums.txt`, and replaces the running binary. Restart the service afterward: `sudo systemctl restart tracelog`.

### Create Admin User

```bash
tracelog user create admin
```

Save the generated password — it's shown only once.

### Start Monitoring

```bash
tracelog serve
```

Open `http://your-server:8090` in your browser and log in.

## Build from Source

```bash
git clone https://github.com/tudorAbrudan/tracelog.git
cd tracelog
make build
./tracelog serve
```

## What's Next?

- [Configuration](/guide/configuration) — Customize ports, bind address, retention
- [Multi-Server Setup](/guide/multi-server) — Monitor multiple servers from one dashboard
- [Alerts](/guide/alerts) — Set up email and webhook notifications
- [Reverse Proxy](/guide/reverse-proxy) — Put TraceLog behind nginx with HTTPS
