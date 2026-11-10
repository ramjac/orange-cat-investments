# Chaos Engineering Runbook

This document details the chaos engineering suite used in the `oci-dev` environment to simulate faults and verify system resilience.

## Chaos Targeting Guards
Fault injection and chaos testing are strictly isolated to `oci-dev` and the external mock instances targeting `oci-dev` in `external-sim` (i.e., components suffixed with `-dev`). Mutating or causing outages in `oci-core` or `oci-internal` environments is forbidden.

The suite runs via `scripts/run-chaos-suite.sh`, which wraps `kubectl` to ensure:
- Only the `oci-dev` namespace is targeted.
- `external-sim` resources are only targeted if they explicitly include the `-dev` suffix.

A robust `trap` ensures that on `EXIT`, `INT`, or `ERR`, all applied chaos policies are deleted and original configurations (like replica counts) are restored.

## Scenarios
The chaos suite encompasses the following test cases:

1. **COTS outage**
   - Applies the `k8s/chaos/oci-dev-deny-internal-egress.yaml` network policy to block egress from `internal-bll` to `oci-internal` systems (e.g., Nextcloud, ERPNext).
   - **Verification:** Employee-portal COTS integrations degrade with clean errors.

2. **Tier partitions**
   - Applies temporary `NetworkPolicy` rules in `oci-dev` that block communication paths between layers (BFF → BLL and BLL → Domain API).
   - **Verification:** Ensures bounded gRPC errors and verifies graceful system recovery upon policy removal.

3. **Database restart**
   - Briefly scales `statefulset/postgres-dev` down to `0` and then back to `1`.
   - **Verification:** Domain APIs reconnect seamlessly without crash loops upon database recovery.

4. **Pod kills**
   - Deletes one pod per running component group (Domain API, BLL, and BFF) in `oci-dev`.
   - **Verification:** The pods are replaced by the Deployment/StatefulSet controller and become `Ready`.

5. **External outage**
   - Applies external faults to the `-dev` mock services in `external-sim`.
   - **Verification:** The overarching simulator runs and asserts proper system functionality under these external faults.

## Expected Alerts
During chaos execution, standard Prometheus alerts tied to application errors, external mock failures, database connectivity, and degraded stateful services will trigger as expected. See the stateful store alerts runbook for specific expected thresholds.
