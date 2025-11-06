# TuserDuser - Multi-Service Event & Feedback Backend

A monorepo containing multiple Go microservices for event management and user feedback collection.

## Project Structure

```text
TuserDuser/
├── .github/
│   └── workflows/
│       └── ci.yml                    # Unified CI/CD pipeline (GitHub Actions)
├── Support/
│   ├── Backend/                      # Feedback & event ingestion service
│   │   ├── main.go
│   │   ├── go.mod
│   │   └── ...
│   ├── proto4.html                   # Frontend demo
│   ├── proto_2.html
│   └── rpoto4.html
├── event-api/                        # Event management REST API
│   ├── main.go
│   ├── go.mod
│   ├── cmd/
│   │   └── server/
│   ├── internal/
│   └── ...
├── Docs/                             # Architecture & documentation
├── bin/                              # Compiled binaries
├── contrib/                          # Deployment scripts
│   ├── deploy_backend.sh             # SSH-based remote deployment
│   ├── install_backend.sh            # Local installation script
│   ├── uninstall_backend.sh          # Uninstall script
│   └── systemd/
│       └── tuserduser-backend.service
├── Makefile                          # Multi-service build orchestration
└── README.md                         # This file
```

## Services

### event-api

**Purpose**: Manage events and provide REST API

See `event-api/README.md` for details.

## Quick Start

### Prerequisites

- Go 1.25+
- macOS (for development) or Linux (for deployment)
- Docker & Docker Compose (optional, for local services)

### Local Development

1. **Clone and navigate**:

   ```bash
   git clone https://github.com/Orange-hanter/TuserDuser.git
   cd TuserDuser
   ```

2. **Build all services**:

   ```bash
   make build                  # Native binaries
   # or
   make build-linux            # Linux/amd64 (for deployment)
   ```

3. **Test services**:

   ```bash
   make test
   ```

## Build Targets

### Single Service Builds

```bash
# Native builds (macOS ARM64, Linux x86_64, etc.)
make build-event-api             # event-api only

# Cross-compile for linux/amd64
make build-linux                 # All services
make build-linux-strip           # All services (stripped)
```

### Code Quality

```bash
make test                        # Run tests
make lint                        # Run linters
make fmt                         # Format code
make vet                         # Run go vet
```

## Deployment

### Prerequisites on Ubuntu Server

- Ubuntu 20.04 LTS or newer
- SSH access with sudo privileges
- Go 1.25+ (or just copy prebuilt binary)

### Option 1: Automated Deployment (Recommended)

From your development machine:

```bash
# Build linux binary first
make build-linux

# Deploy to remote host
./contrib/deploy_backend.sh root@your.server.ip

# Or with custom SSH port and key
./contrib/deploy_backend.sh -p 2222 -i ~/.ssh/id_rsa ubuntu@your.server.ip
```

This script:

1. Copies the binary to the remote host
2. Creates system user `tuserduser`
3. Installs systemd unit
4. Enables and starts the service
5. Copies static files (Support folder)

### Option 2: Manual Installation

On the remote Ubuntu server:

```bash
# Clone or copy repo
git clone https://github.com/Orange-hanter/TuserDuser.git
cd TuserDuser

# Build or copy binary
make build-linux

# Install as systemd service
sudo ./contrib/install_backend.sh

# Start service
sudo systemctl start tuserduser-backend
```

### Option 3: Uninstall

```bash
sudo ./contrib/uninstall_backend.sh
```

## Service Management (Ubuntu/systemd)

Once installed, manage the service with:

```bash
# View status
sudo systemctl status tuserduser-backend

# Start/stop/restart
sudo systemctl start tuserduser-backend
sudo systemctl stop tuserduser-backend
sudo systemctl restart tuserduser-backend

# View logs
sudo journalctl -u tuserduser-backend -f
sudo journalctl -u tuserduser-backend --since "1 hour ago"

# Enable on boot
sudo systemctl enable tuserduser-backend
```

## CI/CD Pipeline

The project uses GitHub Actions (`.github/workflows/ci.yml`) for:

1. **Lint** - golangci-lint, gofmt, go vet
2. **Test** - Unit tests with Postgres & Redis services
3. **Build** - Cross-compile for multiple platforms
4. **Security** - Trivy (vulnerability scanning) + GoSec
5. **Deploy** - Placeholder for production deployment (configure as needed)

Triggers:

- On push to `master`, `main`, `develop` branches
- On pull requests to above branches

## Configuration

### event-api Configuration

See `event-api/README.md` for details.

- Check logs: `sudo journalctl -u tuserduser-backend -f`
- Verify port binding: `lsof -nP -iTCP:8080`
- Ensure Support/rpoto4.html exists in working directory

### "Deploy script fails with SSH errors"

- Verify SSH access: `ssh -p 22 root@your.server.ip 'echo OK'`
- Check key permissions: `chmod 600 ~/.ssh/id_rsa`
- Use verbose mode: `ssh -vv root@your.server.ip ...`

### "Binary not found after deploy"

- Verify file was uploaded: `ssh root@your.server.ip 'ls -l /usr/local/bin/event-api'`
- Check service logs: `sudo journalctl -u event-api -n 50`

## Development Workflow

1. Make changes in `event-api/`
2. Test locally:

   ```bash
   cd event-api && go run ./cmd/server
   ```

3. Run tests & lint:

   ```bash
   make test
   make lint
   ```

4. Commit and push:

   ```bash
   git add .
   git commit -m "your message"
   git push origin your-branch
   ```

5. GitHub Actions will run CI/CD automatically

## Contributing

1. Create a feature branch: `git checkout -b feature/my-feature`
2. Make changes and test locally
3. Push and open a PR
4. CI/CD pipeline will validate
5. Merge after review

## License

See LICENSE file (if present).

## Contact

For issues or questions, open an issue on GitHub.

---

**Last Updated**: November 2025
**Go Version**: 1.25+
**Platforms**: macOS (dev), Linux x86-64 (prod)
