# KataGo MCP Operational Runbooks

This directory contains operational runbooks for deploying, monitoring, and maintaining the KataGo MCP server in production environments.

## Available Runbooks

### Core Operations
- [**Deployment Guide**](deployment.md) - Complete deployment procedures for various environments
- [**Configuration Management**](configuration.md) - Environment-specific configuration guidelines
- [**Monitoring and Alerting**](monitoring.md) - Setting up observability and alerts

### Troubleshooting
- [**Common Issues**](troubleshooting.md) - Frequently encountered problems and solutions
- [**Performance Tuning**](performance-tuning.md) - Optimizing KataGo and server performance
- [**Log Analysis**](log-analysis.md) - Understanding and analyzing logs

### Maintenance
- [**Backup and Recovery**](backup-recovery.md) - Data protection and disaster recovery
- [**Updates and Upgrades**](updates.md) - Safe update procedures
- [**Security Hardening**](security.md) - Security best practices and hardening

### Operations
- [**Scaling Guide**](scaling.md) - Horizontal and vertical scaling strategies
- [**Health Checks**](health-checks.md) - Service health monitoring procedures
- [**Incident Response**](incident-response.md) - Emergency response procedures

## Quick Reference

### Emergency Contacts
- On-call Engineer: [Your on-call system]
- System Administrator: [Your admin contact]
- Security Team: [Your security contact]

### Critical Commands
```bash
# Check service status
systemctl status katago-mcp

# View recent logs
journalctl -u katago-mcp -n 100 --follow

# Emergency restart
systemctl restart katago-mcp

# Check health endpoint
curl http://localhost:8080/health
```

### Key Metrics to Monitor
- Engine uptime and restarts
- Analysis request latency
- Cache hit rate
- Memory usage
- Error rates

## Getting Started

If you're new to operating the KataGo MCP server, start with:

1. [Deployment Guide](deployment.md) - Learn how to deploy the service
2. [Configuration Management](configuration.md) - Understand configuration options
3. [Monitoring and Alerting](monitoring.md) - Set up observability
4. [Common Issues](troubleshooting.md) - Familiarize yourself with common problems

For emergency situations, go directly to [Incident Response](incident-response.md).