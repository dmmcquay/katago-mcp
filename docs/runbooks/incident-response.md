# Incident Response

This guide provides procedures for responding to incidents with the KataGo MCP server.

## Incident Classification

### Severity Levels

#### Critical (P0)
- **Definition**: Complete service outage or security breach
- **Response Time**: Immediate (< 5 minutes)
- **Examples**:
  - Service completely down
  - Data breach or security compromise
  - Corruption affecting data integrity

#### High (P1)
- **Definition**: Major functionality impaired
- **Response Time**: < 30 minutes
- **Examples**:
  - Error rate > 50%
  - Performance degraded > 80%
  - KataGo engine repeatedly failing

#### Medium (P2)
- **Definition**: Partial functionality impaired
- **Response Time**: < 2 hours
- **Examples**:
  - Error rate 10-50%
  - Performance degraded 20-80%
  - Cache not functioning

#### Low (P3)
- **Definition**: Minor issues, workarounds available
- **Response Time**: < 24 hours
- **Examples**:
  - Performance degraded < 20%
  - Non-critical features affected
  - Monitoring issues

## Emergency Response Procedures

### Immediate Response (First 5 Minutes)

1. **Acknowledge the Incident**
   ```bash
   # Acknowledge alerts in monitoring system
   # Update incident tracking system
   ```

2. **Assess Severity**
   ```bash
   # Quick health check
   curl http://localhost:8080/health
   curl http://localhost:8080/ready
   
   # Check service status
   systemctl status katago-mcp
   
   # Check recent logs for obvious errors
   journalctl -u katago-mcp -n 20
   ```

3. **Immediate Mitigation** (if service is down)
   ```bash
   # Quick restart attempt
   sudo systemctl restart katago-mcp
   
   # Check if restart resolved the issue
   sleep 10
   curl http://localhost:8080/health
   ```

### Service Down (P0/P1)

#### Quick Recovery Steps

1. **Service Restart**
   ```bash
   # Standard restart
   sudo systemctl restart katago-mcp
   
   # If that fails, force restart
   sudo systemctl stop katago-mcp
   sudo pkill -f katago-mcp
   sudo pkill -f katago  # Kill any stuck KataGo processes
   sudo systemctl start katago-mcp
   ```

2. **Check for Obvious Issues**
   ```bash
   # Disk space
   df -h
   
   # Memory
   free -h
   
   # Process limits
   ulimit -a
   
   # Configuration syntax
   jq . /etc/katago-mcp/config.json
   ```

3. **Rollback if Recent Changes**
   ```bash
   # If recent deployment, rollback
   sudo cp /backup/katago-mcp-previous /usr/local/bin/katago-mcp
   sudo cp /backup/config.json.backup /etc/katago-mcp/config.json
   sudo systemctl restart katago-mcp
   ```

#### Escalation Triggers

Escalate immediately if:
- Service won't start after restart
- Restart temporarily fixes but service fails again within 5 minutes
- System resources are exhausted
- Configuration appears corrupted

### High Error Rate (P1/P2)

#### Investigation Steps

1. **Check Error Patterns**
   ```bash
   # Recent error logs
   journalctl -u katago-mcp --since "10 minutes ago" | grep -i error
   
   # Error rate metrics
   curl http://localhost:9090/metrics | grep katago_requests_total
   
   # Specific error types
   journalctl -u katago-mcp | grep -E "(timeout|failed|invalid)" | tail -20
   ```

2. **Resource Check**
   ```bash
   # CPU usage
   top -p $(pgrep katago-mcp)
   
   # Memory usage
   ps aux --sort=-%mem | head -10
   
   # KataGo process status
   ps aux | grep katago
   ```

3. **Temporary Mitigation**
   ```bash
   # Reduce load by limiting analysis complexity
   # (requires configuration change and restart)
   
   # Clear cache to free memory
   curl -X POST http://localhost:8080/admin/cache/clear
   
   # Restart KataGo engine if it's stuck
   # (this would be handled by auto-restart mechanism)
   ```

### Performance Degradation (P2/P3)

#### Analysis Steps

1. **Performance Metrics**
   ```bash
   # Check latency percentiles
   curl http://localhost:9090/metrics | grep katago_analysis_duration
   
   # Cache hit rate
   curl http://localhost:9090/metrics | grep katago_cache_hit_rate
   
   # Engine health
   curl http://localhost:9090/metrics | grep katago_engine_up
   ```

2. **Resource Utilization**
   ```bash
   # System load
   uptime
   
   # Memory pressure
   cat /proc/meminfo | grep -E "(MemAvailable|MemFree|Cached)"
   
   # Disk I/O
   iostat -x 1 5
   ```

3. **Optimization Actions**
   ```bash
   # Temporary cache adjustment (if memory pressure)
   # This would require configuration change
   
   # Check for resource contention
   ps aux | grep -E "(high-cpu|memory-intensive)"
   ```

## Communication Procedures

### Internal Communication

#### Incident Declaration
```
INCIDENT DECLARED: [P0/P1/P2/P3]
Service: KataGo MCP Server
Impact: [Brief description]
Started: [timestamp]
Lead: [responder name]
Status page: [if applicable]
```

#### Status Updates (Every 15-30 minutes)
```
UPDATE: KataGo MCP Incident
Time: [timestamp]
Status: [Investigating/Mitigating/Resolved]
Impact: [current impact description]
Actions taken: [bullet points]
Next steps: [what's being done next]
ETA: [if available]
```

### External Communication

#### Customer-Facing Updates
- Use status page if available
- Provide regular updates
- Be transparent about impact
- Avoid technical jargon

#### Template for Status Page
```
[timestamp] - We are investigating reports of issues with the KataGo analysis service. 
Some users may experience timeouts or errors. We are working to resolve this quickly.

[timestamp] - We have identified the issue and are implementing a fix. 
Analysis requests are currently experiencing higher than normal latency.

[timestamp] - The issue has been resolved. All systems are operating normally.
```

## Recovery Procedures

### Data Recovery

If data corruption is suspected:

1. **Stop Service Immediately**
   ```bash
   sudo systemctl stop katago-mcp
   ```

2. **Assess Damage**
   ```bash
   # Check cache integrity
   # Check log files for corruption indicators
   # Verify configuration files
   ```

3. **Restore from Backup**
   ```bash
   # Restore configuration
   sudo cp /backup/config.json.good /etc/katago-mcp/config.json
   
   # Clear potentially corrupted cache
   sudo rm -rf /var/cache/katago-mcp/*
   
   # Restart service
   sudo systemctl start katago-mcp
   ```

### Service Recovery Validation

After any recovery action:

1. **Health Checks**
   ```bash
   # Basic health
   curl http://localhost:8080/health
   curl http://localhost:8080/ready
   
   # Metrics collection
   curl http://localhost:9090/metrics | grep katago_engine_up
   ```

2. **Functional Testing**
   ```bash
   # Test basic analysis (if you have test tools)
   # Monitor for a few minutes to ensure stability
   # Check error rates return to normal
   ```

3. **Performance Validation**
   ```bash
   # Monitor key metrics for 15-30 minutes
   watch "curl -s http://localhost:9090/metrics | grep -E '(katago_analysis_duration|katago_cache_hit_rate)'"
   ```

## Post-Incident Procedures

### Immediate Post-Incident (Within 1 Hour)

1. **Confirm Full Recovery**
   - All metrics back to normal
   - Error rates < 1%
   - Performance within acceptable ranges

2. **Stand Down**
   ```
   INCIDENT RESOLVED: [timestamp]
   Service: KataGo MCP Server
   Duration: [total duration]
   Impact: [final impact summary]
   Resolution: [brief resolution summary]
   ```

3. **Preserve Evidence**
   ```bash
   # Collect logs from incident timeframe
   journalctl -u katago-mcp --since "2 hours ago" > incident-logs.txt
   
   # Collect metrics data
   curl http://localhost:9090/metrics > incident-metrics.txt
   
   # System state during incident
   ps aux > incident-processes.txt
   free -h > incident-memory.txt
   df -h > incident-disk.txt
   ```

### Post-Incident Review (Within 48 Hours)

#### Root Cause Analysis

1. **Timeline Construction**
   - When did the incident start?
   - What triggered it?
   - When was it detected?
   - What actions were taken?
   - When was it resolved?

2. **Impact Assessment**
   - Duration of impact
   - Number of affected requests
   - Customer impact
   - Revenue impact (if applicable)

3. **Root Cause Identification**
   - What was the underlying cause?
   - Why wasn't it caught earlier?
   - What allowed it to happen?

#### Action Items

Template for action items:
```
1. [Action description]
   Owner: [person]
   Due date: [date]
   Priority: [High/Medium/Low]
   
2. Improve monitoring for [specific condition]
   Owner: [SRE team]
   Due date: [2 weeks]
   Priority: High
```

## Escalation Procedures

### When to Escalate

#### To Senior Engineer
- Unable to restart service within 10 minutes
- Issue returns after attempted fix
- Unfamiliar error patterns

#### To Development Team
- Suspected application bug
- Need code-level investigation
- Performance issues requiring code changes

#### To Security Team
- Suspected security breach
- Unusual access patterns
- Data integrity concerns

### Escalation Contacts

```bash
# Update these with your actual contacts
ONCALL_ENGINEER="oncall@company.com"
SENIOR_SRE="senior-sre@company.com"
DEV_TEAM_LEAD="dev-lead@company.com"
SECURITY_TEAM="security@company.com"
MANAGER="manager@company.com"
```

### Escalation Template

```
ESCALATION NEEDED: KataGo MCP Incident

Incident ID: [ID]
Severity: [P0/P1/P2/P3]
Duration: [how long has this been going on]
Current responder: [name]

Summary:
[Brief description of the issue]

Impact:
[What is currently broken/degraded]

Actions taken:
- [List what has been tried]
- [Include timestamps]

Why escalating:
[Why you need help - stuck, need expertise, etc.]

Next person: [who you're escalating to]
```

## Prevention Measures

### Proactive Monitoring
- Comprehensive alerting on all key metrics
- Regular health checks
- Capacity monitoring and planning

### Regular Testing
- Disaster recovery drills
- Load testing
- Chaos engineering exercises

### Documentation Maintenance
- Keep runbooks updated
- Regular review of procedures
- Training for new team members

### Automation
- Automated recovery for common issues
- Self-healing systems where possible
- Automated alerting and escalation

## Training and Drills

### Monthly Incident Response Drill

1. **Scenario**: Simulate a P1 incident
2. **Participants**: On-call engineer + backup
3. **Evaluation**: Response time, actions taken, communication
4. **Improvement**: Update procedures based on learnings

### Quarterly Review

- Review all incidents from quarter
- Identify trends and patterns
- Update procedures and training
- Review escalation contacts and procedures

## Next Steps

- Implement [monitoring and alerting](monitoring.md) to detect issues early
- Set up [backup and recovery](backup-recovery.md) procedures
- Review [troubleshooting guide](troubleshooting.md) for common issues