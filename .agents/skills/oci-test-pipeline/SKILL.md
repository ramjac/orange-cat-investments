---
name: oci-test-pipeline
description: >-
  Verifies and tests the current state of Orange Cat Investments (OCI) code:
  runs the full test suite with `go test -v -race ./...`, fixes any issues without
  getting sidetracked, deploys to the test Kubernetes cluster if available, and
  executes the platform simulator. Use when asked to test OCI code, verify platform
  health, or validate changes end-to-end.
---

# OCI Code Verification & Test Pipeline

Use this skill to verify the current state of the OCI codebase. Follow these sequential steps strictly and **do not get sidetracked**.

---

## Guiding Rules (Stay Focused)

1. **Do not get sidetracked:** Focus exclusively on the testing, deployment, and simulation sequence. Do not refactor unrelated components, reformat clean files, or pursue tangential ideas.
2. **Always use the exact test command:** Always run the complete test suite with `go test -v -race ./...`. Do not switch to testing individual sub-packages or running ad-hoc command variants with custom flags.
3. **No rabbit holes on cluster outages:** If the test Kubernetes cluster is offline or unreachable, record the status and proceed immediately to simulation.

---

## Step 1: Run Full Go Test Suite with Race Detection

Run the complete test suite across the monorepo:

```bash
go test -v -race ./...
```

### If Any Failures Occur:
1. Examine the failure output (logical failure, panic, compilation error, or `DATA RACE`).
2. Apply a targeted, minimal fix directly to the affected code.
3. Re-run `go test -v -race ./...` to verify the fix and ensure no regressions were introduced.
4. Do not proceed to Step 2 until `go test -v -race ./...` passes cleanly.

---

## Step 2: Deploy to Test Kubernetes (If Available)

Check whether the test Kubernetes environment (such as `thinker`) is accessible before deploying.

### 1. Check Cluster Connectivity
```bash
if [ -f "thinker-kube.config" ]; then
  export KUBECONFIG="$(pwd)/thinker-kube.config"
fi
kubectl cluster-info --request-timeout=3s
```

### 2. Deployment Action:
* **If cluster is reachable:**
  Deploy the platform workloads:
  ```bash
  ./scripts/deploy-k8s.sh
  ```
  Check that core pods are running:
  ```bash
  kubectl get pods -n oci-core
  ```
* **If cluster is unreachable or kubectl is not found:**
  Note that the test cluster is unavailable and proceed directly to Step 3. Do not spend time attempting cluster repairs or debugging remote networking.

---

## Step 3: Run the Simulator

Validate persona interactions and multi-tier platform flows with the simulator driver:

* **If Kubernetes was deployed and reachable:**
  Run against the staging ingress:
  ```bash
  go run ./simulation/cmd/simulator -mode live -url http://dev.api.oci.local
  ```
* **If cluster is unavailable (or for standalone verification):**
  Run in mock mode:
  ```bash
  go run ./simulation/cmd/simulator -mode mock -iterations 1
  ```

### Verify Results
Confirm that all 12 corporate, caretaker, investor, and developer personas execute their scheduled actions cleanly without runtime panics or unexpected errors.

---

## Final Report

Provide a brief, structured summary:
- **Go Race Test Status:** Pass/Fail (and details on any fixes made).
- **Kubernetes Deployment:** Deployed successfully or skipped (cluster unreachable).
- **Simulator Status:** Pass/Fail across personas.
