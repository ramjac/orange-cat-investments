# MEMORY - Orange Cat Investments (OCI)

## Test Pipeline Commands (see .agents/skills/oci-test-pipeline/SKILL.md)

1. Tests: `go test -v -race ./...` (always the full monorepo, never sub-packages)
2. Deploy: `export KUBECONFIG="$(pwd)/thinker-kube.config" && ./scripts/deploy-k8s.sh`
   - Thinker cluster: MicroK8s at 192.168.1.246 (kubeconfig: `thinker-kube.config`, gitignored)
   - After deploy, wait for `oci-dev` deployments to roll out before simulating
3. Simulator (run from this workstation, never on thinker host):
   ```bash
   OCI_INGRESS_IP=192.168.1.246 \
   FORGEJO_ELISE_TOKEN=<pat> FORGEJO_MARCUS_TOKEN=<pat> \
   go run ./simulation/cmd/simulator -mode live -url http://dev.api.oci.local
   ```
   - `OCI_INGRESS_IP` is required: the sim client dials `*.oci.local` hosts at that IP
     (see docs/runbooks/cluster-load-testing.md). Without it, DNS fails from the workstation.
   - Do NOT edit /etc/hosts.

## Forgejo Persona Tokens (for simulation)

- In-cluster Forgejo users: `emp-human-elise`, `emp-human-marcus` (plus admin `oci-admin`)
- Generate a PAT (in-cluster, no API auth needed):
  ```bash
  kubectl exec -n oci-internal deployment/forgejo -- \
    forgejo admin user generate-access-token -u emp-human-elise -t <name> --raw --scopes all
  ```
  - MUST pass `--scopes all`: omitting it stores an EMPTY scope, which Forgejo's
    scope enforcement treats as no permissions (403 "token does not have at least
    one of required scope(s)").
- Export the result as `FORGEJO_ELISE_TOKEN` / `FORGEJO_MARCUS_TOKEN` when running the sim.
- PATs persist in postgres-internal across cluster restarts; regenerate if 401s appear.

## Simulation Failure Triage

- `insufficient cash balance` / `insufficient shares to sell` from the mock broker:
  the `mock-broker` pod (external-sim) holds an in-memory ledger seeded at startup
  (`cmd/mock-broker/ledger/seed.go`) that drifts across repeated sim runs.
  Fix: `kubectl rollout restart deployment/mock-broker -n external-sim`, then re-run.
- Forgejo `token is required` (401): missing/expired persona PATs (see above).
