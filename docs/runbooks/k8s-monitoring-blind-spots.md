# Kubernetes Monitoring Blind Spots Runbook

This guide covers strategies for addressing infrastructure monitoring blind spots in a self-hosted bare-metal Kubernetes environment.

## 1. Kubernetes API Event Shipping to Long-Term Logs

By default, the Kubernetes API server purges cluster events after 60 minutes. To maintain an audit trail and aid in debugging intermittent failures, these events should be shipped to a long-term logging system.

### Implementation Strategy with Grafana Promtail / Loki

We can use Grafana Promtail's `kubernetes_sd_configs` or Grafana Alloy to tail Kubernetes events.
A robust implementation leverages Promtail or Alloy configured as a Deployment that authenticates with the Kubernetes API, watches the `/api/v1/events` endpoint, and pushes these events into Loki.

Example Promtail snippet (if using `loki.source.kubernetes_events` in Alloy):
```alloy
loki.source.kubernetes_events "k8s_events" {
  job_name   = "kubernetes-events"
  log_format = "json"
  forward_to = [loki.write.default.receiver]
}
```

## 2. "Watching the Watchers" (Dead Man's Switch)

If the cluster loses power or network connectivity, internal Prometheus and Alertmanager instances cannot send alerts. An external Dead Man's Switch is necessary.

### Setup Instructions

1.  **Configure Watchdog Alert**: Prometheus Alertmanager comes with a built-in `Watchdog` alert that fires continuously.
2.  **External Monitoring Service**: Sign up for a service like [Healthchecks.io](https://healthchecks.io), Better Stack, or Cronitor.
3.  **Alertmanager Receiver**: Configure an Alertmanager receiver to send a webhook to the external service URL.
4.  **Routing**: Set up a route in Alertmanager to send the `Watchdog` alert to the new receiver every 5-10 minutes.

Example `alertmanager.yml` snippet:
```yaml
route:
  receiver: 'default-receiver'
  routes:
    - match:
        alertname: Watchdog
      receiver: 'dead-mans-switch'
      repeat_interval: 5m

receivers:
  - name: 'dead-mans-switch'
    webhook_configs:
      - url: 'https://hc-ping.com/your-uuid-here'
```
If the external service stops receiving pings within its grace period (e.g., 15 minutes), it will alert the on-call engineer via an external channel (SMS/Email).

## 3. Automated Backup Verification

Stateful applications require daily backups. Monitoring the backup controller metrics is crucial to ensure these backups are successful.

### Implementation Strategy

If using a custom Kubernetes CronJob for backups, the script should use the Prometheus Pushgateway (or write a metric text file exposed via a node-exporter `textfile` collector) to emit the timestamp of the last successful backup.

Example bash snippet inside a Backup CronJob:
```bash
if [ $BACKUP_EXIT_CODE -eq 0 ]; then
  # Push to Pushgateway
  echo "backup_last_successful_timestamp_seconds{job=\"postgres-backup\"} $(date +%s)" | curl --data-binary @- http://pushgateway.monitoring.svc.cluster.local:9091/metrics/job/postgres-backup
fi
```
The alert `BackupFailedOrStale` (already configured in Prometheus) will fire if this metric is older than 26 hours.
