# ADR 0005: Software Supply Chain Security, Distroless Runtimes, Harbor & Kyverno Admission Control

* **Status:** Accepted
* **Date:** 2026-11-22
* **Author:** Frank Operations (`emp-human-frank`), Platform Security & K8s Infrastructure Lead & Marcus Lead (`emp-human-marcus`), Principal Systems Architect
* **Context:** Architectural specification for Roadmap Phase 8 (`7.30`–`7.35`), hardening the OCI software supply chain from source commit through container admission in `oci-core` and `oci-dev`. Evolves the Alpine packaging model originally established in [ADR 0003](0003-gitops-containerization-promotion-pipeline.md).

---

## 1. Context and Problem Statement

While ADR 0003 introduced multi-stage builds and an automated staging gate in `oci-dev`, five supply-chain and runtime security risks remained:
1. **Unenforced Architectural & Security Invariants:** Domain isolation (`AGENTS.md` §3), vendor-identifier encapsulation, secret hygiene, and Go security linting relied on manual code review rather than blocking CI gates.
2. **Writable Shells & Package Managers in Runtime Containers:** `alpine:3.20` base images included `/bin/sh` and `apk`, expanding post-exploitation blast radius.
3. **Barebones Container Registry Without Vulnerability Scanning:** `k8s/registry/registry.yaml` (`registry:2`) lacked UI governance, OIDC authentication, pull-through caching, or automated CVE gating.
4. **Unsigned Container Images & Edge Firmware Blobs:** Neither container images nor OTA firmware binaries distributed to edge devices carried cryptographic provenance or CycloneDX SBOMs.
5. **No Cluster Admission Enforcement:** Nothing at the Kubernetes API server prevented `:latest` tags, privileged containers, unsigned images, or TLS `subPath` mounts from being scheduled into `oci-core`.

---

## 2. Decision Outcome

We adopt an end-to-end **Zero-Trust Software Supply Chain & Admission Control Architecture**:

1. **Static Architectural & Security Gates (`7.30`):**
   * Added [`.semgrep.yml`](../../.semgrep.yml) mechanically enforcing BFF isolation, cross-domain API isolation, vendor identifier encapsulation, and zero hardcoded personas in `oci-core` code.
   * Enabled `golangci-lint` v2 (with `gosec`), `govulncheck ./...`, and `gitleaks detect` in [`.forgejo/workflows/ci.yaml`](../../.forgejo/workflows/ci.yaml) and [`scripts/verify-ci-contracts.sh`](../../scripts/verify-ci-contracts.sh).
2. **Distroless Runtime Images & Restricted Pod Security Standards (`7.31`):**
   * Migrated all 13 Go service Dockerfiles (`cmd/*/Dockerfile`) and `web/marketing/Dockerfile` (`containerPort: 8080`) to `gcr.io/distroless/static-debian12:nonroot` (`USER nonroot:nonroot`).
   * Enforced `runAsNonRoot: true`, `runAsUser: 65532`, `readOnlyRootFilesystem: true`, `allowPrivilegeEscalation: false`, and `capabilities.drop: ["ALL"]` across all Go Deployments and CronJobs.
3. **Forgejo Package Registry for OTA Firmware (`7.32`):**
   * Enabled `FORGEJO__packages__ENABLED=true` and added [`scripts/publish-firmware-package.sh`](../../scripts/publish-firmware-package.sh) to publish versioned `.bin` firmware and `.pbw` watch bundles.
4. **Harbor Enterprise Container Registry & Trivy Gate (`7.33`):**
   * Replaced `registry:2` with Harbor ([`k8s/registry/harbor.yaml`](../../k8s/registry/harbor.yaml)) on `:32000` / `registry.oci.local` featuring Forgejo OIDC auth, pull-through proxy caches (`proxy-dockerhub`, `proxy-ghcr`, `proxy-gcr`), and an embedded Trivy scanner.
   * Added `trivy image --exit-code 1 --severity CRITICAL,HIGH` to CI before image push.
5. **CycloneDX SBOMs & Cryptographic Cosign Signing (`7.34`):**
   * CI generates CycloneDX SBOMs (`syft`) and signs/attests all 16 container images (`cosign sign`, `cosign attest`).
   * Added `cosign_signature` to `facilities.firmware_releases` and implemented [`pkg/cosignverify`](../../pkg/cosignverify/verify.go) so `oci-worker` and edge devices (`cmd/mock-hardware/ota_verify.go`) reject unsigned or tampered firmware against `ConfigMap/oci-cosign-pub`.
6. **Kyverno Cluster Admission Control (`7.35`):**
   * Deployed Kyverno ([`k8s/security/kyverno.yaml`](../../k8s/security/kyverno.yaml)) and four `Enforce`-mode `ClusterPolicy` rules ([`k8s/security/kyverno-policies.yaml`](../../k8s/security/kyverno-policies.yaml)): `verify-oci-image-signatures`, `disallow-latest-tag`, `enforce-restricted-pss`, and `forbid-tls-subpath-mounts`, verified against negative fixtures ([`invalid-manifests.yaml`](../../k8s/security/fixtures/invalid-manifests.yaml)).

---

## 3. Consequences

* **Tamper-Resistant Releases:** Every production pod in `oci-core` runs an immutable `:sha-<COMMIT>` Distroless image verified by Trivy, accompanied by a CycloneDX SBOM, and signed by Cosign.
* **Defense Against Misconfiguration:** Kyverno rejects non-compliant manifests at Kubernetes admission time, preventing drift from `AGENTS.md` invariants.
