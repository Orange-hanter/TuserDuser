# Production Deployment Summary

## ✅ Deployment Complete

### 📋 Server Configuration

**Server:** tuserduser.com (Ubuntu 24.04)

- **IP:** (use `ssh tuser` to connect)
- **User:** root (via SSH key)
- **RAM:** 961MB (637MB available)
- **Disk:** 15GB (9.6GB free)

### 🎯 Deployed Services

#### Event API Service

- **Status:** ✅ Running
- **Location:** `/opt/event-api/`
- **Port:** 8081 (internal)
- **Service:** `event-api.service`
- **User:** `eventapi`

#### PostgreSQL 16

- **Status:** ✅ Running
- **Database:** `event_api`
- **User:** `eventapi`
- **Port:** 5432

#### Redis 7

- **Status:** ✅ Running
- **Port:** 6379
- **Password:** Protected

#### Nginx

- **Status:** ✅ Running
- **Config:** `/etc/nginx/sites-available/event-api-backend`
- **Logs:** `/var/log/nginx/event-api-*.log`

### 🔗 Endpoints

Once DNS is configured for `api.tuserduser.online`:

- **API Base:** `http://api.tuserduser.online/v1/api/`
- **Health:** `http://api.tuserduser.online/v1/api/health`
- **Swagger:** `http://api.tuserduser.online/swagger/index.html`

**Current (direct server):**

- Health: `http://localhost:8081/health` (from server)
- API: `http://localhost:8081/v1/api/...`

### 📁 Directory Structure

````text
/opt/event-api/
├── bin/
│   └── event-api           # Go binary (24MB)
├── logs/
│   ├── event-api.log       # Application logs
│   ├── event-api-error.log # Error logs
│   └── backup.log          # Backup logs
├── backups/                # Database backups
├── .env                    # Environment configuration
└── backup.sh               # Backup script
```bash
### 🔐 Credentials

All credentials are stored in `/opt/event-api/.env`:

- **Database Password:** Auto-generated (32 chars)
- **Redis Password:** Auto-generated (32 chars)
- **JWT Secret:** Auto-generated (64 chars)

**Note:** SMTP settings need to be configured manually in `.env`

### 🚀 Management Commands

#### Service Management

```bash
## Status
## Status
sudo systemctl status event-api

## Start/Stop/Restart
## Start/Stop/Restart
sudo systemctl start event-api
sudo systemctl stop event-api
sudo systemctl restart event-api

## View logs (live)
## View logs (live)
sudo tail -f /opt/event-api/logs/event-api.log
````

#### Database Management

````bash
## Backup
## Backup
sudo -u eventapi /opt/event-api/backup.sh

## Connect to database
## Connect to database
sudo -u postgres psql -d event_api

## Check backups
## Check backups
ls -lh /opt/event-api/backups/
```bash
#### Deployment (from local machine)

```bash
## Build and deploy
## Build and deploy
./scripts/deploy-binary.sh

## Or manually:
## Or manually:
GOOS=linux GOARCH=amd64 go build -o bin/event-api-linux ./cmd/server
scp bin/event-api-linux tuser:/tmp/event-api
ssh tuser "sudo systemctl stop event-api && sudo mv /tmp/event-api /opt/event-api/bin/event-api && sudo chown eventapi:eventapi /opt/event-api/bin/event-api && sudo chmod +x /opt/event-api/bin/event-api && sudo systemctl start event-api"
````

### 📊 Resource Usage

Current memory usage:

- PostgreSQL: ~30-50MB
- Redis: ~10-20MB
- Event API: ~11MB
- **Total:** ~60-80MB (plenty of headroom)

### 🔧 Configuration Files

#### Systemd Service

`/etc/systemd/system/event-api.service`

- Auto-restart on failure
- Runs as `eventapi` user
- Logs to `/opt/event-api/logs/`

#### Nginx Configuration

`/etc/nginx/sites-available/event-api-backend`

- Rate limiting: 10 req/s (burst 20)
- Upstream: localhost:8081
- Gzip compression enabled
- Security headers added

#### Environment Variables

`/opt/event-api/.env`

- Port: 8081
- Environment: production
- CORS origins configured
- Database and Redis credentials

### 📝 Next Steps

1. **Configure DNS**

   ```bash
   # Add A record:
   api.tuserduser.online -> <server_ip>
   ```

2. **Setup SSL Certificate**

   ```bash
   ssh tuser
   sudo apt-get install -y certbot python3-certbot-nginx
   sudo certbot --nginx -d api.tuserduser.online
   ```

3. **Configure SMTP** (for email verification)

   ```bash
   ssh tuser
   sudo nano /opt/event-api/.env
   # Update SMTP_* variables
   sudo systemctl restart event-api
   ```

4. **Configure SMS** (optional)

   ```bash
   # Update SMS_PROVIDER, SMS_API_KEY in .env
   # Options: smsru, smsc, twilio
   ```

5. **Test API**

   ```bash
   # Registration
   curl -X POST http://api.tuserduser.online/v1/api/auth/register \
     -H "Content-Type: application/json" \
     -d '{"email":"test@example.com","phone":"+79991234567","password":"password123"}'

   # Health check
   curl http://api.tuserduser.online/v1/api/health
   ```

6. **Monitor Logs**

   ```bash
   # Application logs
   ssh tuser sudo tail -f /opt/event-api/logs/event-api.log

   # Nginx logs
   ssh tuser sudo tail -f /var/log/nginx/event-api-access.log

   # System logs
   ssh tuser sudo journalctl -u event-api -f
   ```

### 🔒 Security Checklist

- [x] Application runs as non-root user (`eventapi`)
- [x] Database password auto-generated (32 chars)
- [x] Redis password protected
- [x] JWT secret auto-generated (64 chars)
- [x] Nginx rate limiting enabled
- [x] Environment file permissions: 600
- [ ] SSL/TLS certificate (pending DNS)
- [ ] Firewall configured
- [ ] Fail2ban for SSH protection

### 🔄 Automated Backups

Daily database backups are scheduled via cron:

- **Time:** 2:00 AM daily
- **Location:** `/opt/event-api/backups/`
- **Retention:** 7 days (auto-cleanup)
- **Log:** `/opt/event-api/logs/backup.log`

Manual backup:

````bash
ssh tuser sudo -u eventapi /opt/event-api/backup.sh
```bash
### 📈 Monitoring

**Check service status:**

```bash
ssh tuser "sudo systemctl is-active event-api && echo 'API is running' || echo 'API is DOWN'"
````

**Check health endpoint:**

````bash
ssh tuser "curl -f http://localhost:8081/health && echo 'Healthy' || echo 'Unhealthy'"
```bash
**Resource usage:**

```bash
ssh tuser "free -h && df -h / && ps aux | grep event-api"
````

### 🐛 Troubleshooting

**Service not starting:**

````bash
## Check logs
## Check logs
sudo journalctl -u event-api -n 50

## Check application logs
## Check application logs
sudo tail -50 /opt/event-api/logs/event-api-error.log

## Check configuration
## Check configuration
sudo -u eventapi /opt/event-api/bin/event-api --help
```bash
**Database connection issues:**

```bash
## Test connection
## Test connection
sudo -u postgres psql -d event_api -c "SELECT version();"

## Check PostgreSQL status
## Check PostgreSQL status
sudo systemctl status postgresql
````

**Redis connection issues:**

````bash
## Test Redis
## Test Redis
redis-cli -a $(sudo grep REDIS_PASSWORD /opt/event-api/.env | cut -d= -f2) ping

## Check Redis status
## Check Redis status
sudo systemctl status redis-server
```bash
**Port conflicts:**

```bash
## Check what's using port 8081
## Check what's using port 8081
sudo lsof -i :8081

## Change port in .env if needed
## Change port in .env if needed
sudo nano /opt/event-api/.env  # Update PORT=
sudo systemctl restart event-api
````

### 📚 Documentation

- [CI/CD Documentation](../CI_CD.md)
- [SMS Service](../SMS_SERVICE.md)
- [API Documentation](../API_DOCUMENTATION.md)
- [GitHub Secrets Setup](../GITHUB_SECRETS_SETUP.md)

### 🎉 Success

Your Event API is now running in production!

**Quick health check:**

```bash
ssh tuser "curl -s http://localhost:8081/health && systemctl is-active event-api"
```

Should return: `OK` and `active`
