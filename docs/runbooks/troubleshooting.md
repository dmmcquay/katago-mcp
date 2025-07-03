# Troubleshooting Guide

This guide covers common issues and their solutions when operating the KataGo MCP server.

## Quick Diagnosis Commands

```bash
# Service status
systemctl status katago-mcp

# Recent logs
journalctl -u katago-mcp -n 50 --follow

# Health check
curl http://localhost:8080/health

# Resource usage
ps aux | grep katago
free -h
df -h

# Network connections
netstat -tulnp | grep katago-mcp
```

## Common Issues

### 1. Service Won't Start

#### Symptoms
- `systemctl start katago-mcp` fails
- Service shows "failed" status
- No process running

#### Diagnosis
```bash
# Check service status
systemctl status katago-mcp

# Check logs for startup errors
journalctl -u katago-mcp -n 100

# Validate configuration
katago-mcp --config /etc/katago-mcp/config.json --validate

# Check file permissions
ls -la /etc/katago-mcp/
ls -la /usr/local/bin/katago-mcp
```

#### Common Causes & Solutions

**Configuration File Issues**:
```bash
# Check JSON syntax
jq . /etc/katago-mcp/config.json

# Fix common configuration errors:
# - Invalid JSON syntax
# - Missing required fields
# - Incorrect file paths
```

**Permission Issues**:
```bash
# Fix binary permissions
sudo chmod +x /usr/local/bin/katago-mcp

# Fix config permissions
sudo chown katago-mcp:katago-mcp /etc/katago-mcp/config.json
sudo chmod 640 /etc/katago-mcp/config.json
```

**Missing Dependencies**:
```bash
# Check KataGo installation
which katago
katago version

# Check model file exists
ls -la /opt/katago/models/

# Verify config file exists
ls -la /opt/katago/config/analysis.cfg
```

### 2. KataGo Engine Fails to Start

#### Symptoms
- Service starts but KataGo engine is down
- `katago_engine_up` metric shows 0
- "Failed to start engine" in logs

#### Diagnosis
```bash
# Check KataGo directly
sudo -u katago-mcp katago analysis \
  -config /opt/katago/config/analysis.cfg \
  -model /opt/katago/models/your-model.bin.gz

# Check file access
sudo -u katago-mcp ls -la /opt/katago/models/
sudo -u katago-mcp ls -la /opt/katago/config/
```

#### Solutions

**Model File Issues**:
```bash
# Re-download corrupted model
cd /opt/katago/models/
sudo rm your-model.bin.gz
sudo wget https://media.katagoarchive.org/your-model.bin.gz

# Verify model integrity
sudo katago benchmark -model your-model.bin.gz -visits 1
```

**Configuration Issues**:
```bash
# Regenerate KataGo config
cd /opt/katago/config/
sudo katago genconfig -model ../models/your-model.bin.gz -output analysis.cfg

# Test configuration
sudo katago analysis -config analysis.cfg -model ../models/your-model.bin.gz < /dev/null
```

**Resource Constraints**:
```bash
# Check available memory
free -h

# Reduce threads in katago config
sudo nano /opt/katago/config/analysis.cfg
# Set: numSearchThreads = 1

# Reduce model size (use smaller model)
```

### 3. High Memory Usage

#### Symptoms
- OOM kills in system logs
- Service becomes unresponsive
- High memory alerts

#### Diagnosis
```bash
# Check memory usage
ps aux --sort=-%mem | head -10
cat /proc/$(pgrep katago-mcp)/status | grep -E "(VmRSS|VmSize)"

# Check cache usage
curl http://localhost:9090/metrics | grep katago_cache

# Check system memory
free -h
cat /proc/meminfo
```

#### Solutions

**Reduce Cache Size**:
```json
{
  "cache": {
    "enabled": true,
    "maxItems": 500,
    "maxSizeBytes": 52428800,
    "ttlSeconds": 1800
  }
}
```

**Optimize KataGo Settings**:
```bash
# Reduce KataGo memory usage in analysis.cfg
sudo nano /opt/katago/config/analysis.cfg

# Key settings to reduce:
# nnCacheSizePowerOfTwo = 16  # (was 20, reduces neural net cache)
# numSearchThreads = 2       # (reduce parallel search threads)
# maxVisits = 500            # (reduce analysis depth)
```

**Implement Memory Limits**:
```bash
# Add to systemd service
sudo systemctl edit katago-mcp

# Add:
[Service]
MemoryMax=4G
MemorySwapMax=0
```

### 4. Slow Response Times

#### Symptoms
- High latency alerts
- Client timeouts
- Poor user experience

#### Diagnosis
```bash
# Check analysis latency metrics
curl http://localhost:9090/metrics | grep katago_analysis_duration

# Test direct KataGo performance
time echo '{"id":"test","boardXSize":19,"boardYSize":19,"rules":"tromp-taylor","komi":7.5,"moves":[]}' | \
  katago analysis -config /opt/katago/config/analysis.cfg -model /opt/katago/models/your-model.bin.gz

# Check cache hit rate
curl http://localhost:9090/metrics | grep katago_cache_hit_rate

# Monitor CPU usage
top -p $(pgrep katago)
```

#### Solutions

**Optimize KataGo Settings**:
```bash
# Balance speed vs strength in analysis.cfg
numSearchThreads = 4        # Increase for faster analysis
maxPlayouts = 200          # Reduce for faster analysis
maxTime = 5.0              # Set time limit
maxVisits = 800            # Reduce for faster analysis
```

**Improve Cache Performance**:
```json
{
  "cache": {
    "enabled": true,
    "maxItems": 2000,         
    "maxSizeBytes": 209715200,
    "ttlSeconds": 7200        
  }
}
```

**Hardware Optimization**:
- Use faster CPU
- Increase CPU cores
- Use SSD storage
- Consider GPU acceleration (if KataGo supports it)

### 5. High Error Rates

#### Symptoms
- Error rate alerts
- Failed analysis requests
- Client errors

#### Diagnosis
```bash
# Check error metrics
curl http://localhost:9090/metrics | grep katago_requests_total

# Analyze error logs
journalctl -u katago-mcp | grep -i error | tail -20

# Check specific error types
journalctl -u katago-mcp --since "1 hour ago" | \
  grep -E "(timeout|failed|error)" | sort | uniq -c
```

#### Common Error Types & Solutions

**Timeout Errors**:
```json
{
  "katago": {
    "maxTime": 15.0,          
    "maxVisits": 1000         
  }
}
```

**Invalid SGF/Position Errors**:
- Check input validation
- Verify SGF parsing
- Review move validation logic

**Engine Restart Errors**:
```bash
# Check for engine instability
journalctl -u katago-mcp | grep "engine restart"

# Monitor engine health
watch "curl -s http://localhost:9090/metrics | grep katago_engine_up"
```

### 6. Cache Issues

#### Symptoms
- Low cache hit rates
- Memory pressure from cache
- Inconsistent cache behavior

#### Diagnosis
```bash
# Check cache statistics
curl http://localhost:9090/metrics | grep katago_cache

# Monitor cache size growth
watch "curl -s http://localhost:9090/metrics | grep katago_cache_size_bytes"

# Check TTL expiration patterns
journalctl -u katago-mcp | grep "Cache entry expired"
```

#### Solutions

**Optimize Cache Configuration**:
```json
{
  "cache": {
    "enabled": true,
    "maxItems": 1500,          
    "maxSizeBytes": 157286400, 
    "ttlSeconds": 5400         
  }
}
```

**Cache Key Issues**:
- Verify cache key generation
- Check for key collisions
- Review cache invalidation logic

### 7. Network Issues

#### Symptoms
- Connection timeouts
- Service unreachable
- Port binding errors

#### Diagnosis
```bash
# Check port availability
netstat -tulnp | grep ":8080\|:9090"

# Test connectivity
curl -v http://localhost:8080/health
telnet localhost 8080

# Check firewall
sudo ufw status
iptables -L -n
```

#### Solutions

**Port Conflicts**:
```bash
# Find what's using the port
sudo lsof -i :8080
sudo fuser 8080/tcp

# Change port in configuration
```

**Firewall Issues**:
```bash
# Allow ports through firewall
sudo ufw allow 8080
sudo ufw allow 9090

# Check iptables rules
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
```

## Performance Debugging

### 1. CPU Profiling

```bash
# Install pprof tools
go install github.com/google/pprof@latest

# If service has pprof endpoints enabled
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
pprof cpu.prof
```

### 2. Memory Profiling

```bash
# Analyze memory usage
curl http://localhost:6060/debug/pprof/heap > heap.prof
pprof heap.prof

# Check for memory leaks
curl http://localhost:6060/debug/pprof/allocs > allocs.prof
```

### 3. Trace Analysis

```bash
# Capture execution trace
curl http://localhost:6060/debug/pprof/trace?seconds=5 > trace.out
go tool trace trace.out
```

## Recovery Procedures

### 1. Service Recovery

```bash
# Immediate restart
sudo systemctl restart katago-mcp

# Force stop and restart
sudo systemctl stop katago-mcp
sudo pkill -f katago-mcp
sudo systemctl start katago-mcp

# Reset to known good state
sudo systemctl stop katago-mcp
sudo cp /backup/config.json.good /etc/katago-mcp/config.json
sudo systemctl start katago-mcp
```

### 2. Emergency Procedures

**High Load Mitigation**:
```bash
# Temporarily reduce KataGo resources
sudo systemctl edit katago-mcp --full
# Reduce numThreads, maxVisits in config

# Clear cache to free memory
curl -X POST http://localhost:8080/admin/cache/clear
```

**Resource Exhaustion**:
```bash
# Free up disk space
sudo journalctl --vacuum-time=7d
sudo docker system prune -a

# Kill memory-intensive processes
sudo pkill -f "high-memory-process"
```

## Escalation Procedures

### When to Escalate

1. **Critical**: Service completely down for >5 minutes
2. **High**: Error rate >20% for >10 minutes  
3. **Medium**: Performance degraded >50% for >30 minutes
4. **Security**: Potential security breach detected

### Escalation Contacts

1. **Level 1**: On-call engineer
2. **Level 2**: Senior SRE team
3. **Level 3**: Development team lead
4. **Security**: Security incident response team

### Information to Gather

```bash
# System state
systemctl status katago-mcp
ps aux | grep katago
free -h
df -h

# Service logs (last 1000 lines)
journalctl -u katago-mcp -n 1000 > /tmp/katago-mcp.log

# Metrics snapshot
curl http://localhost:9090/metrics > /tmp/metrics.txt

# Configuration
cp /etc/katago-mcp/config.json /tmp/

# Network state
netstat -tulnp > /tmp/netstat.txt
```

## Prevention

### 1. Proactive Monitoring

- Set up comprehensive alerting
- Regular health checks
- Capacity planning
- Performance baseline monitoring

### 2. Regular Maintenance

- Log rotation
- Cache cleanup
- Configuration reviews
- Security updates

### 3. Testing

- Load testing
- Failover testing
- Configuration validation
- Disaster recovery drills

## Next Steps

For specific scenarios:
- [Performance Tuning](performance-tuning.md) for optimization
- [Log Analysis](log-analysis.md) for detailed investigation
- [Incident Response](incident-response.md) for emergency procedures