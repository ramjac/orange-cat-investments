# ADR 0006: Zero-Trust Networking, Kubernetes Gateway API & Coraza Edge WAF

* **Status:** Accepted
* **Date:** 2026-11-22
* **Author:** Frank Operations (`emp-human-frank`), Platform Security & K8s Infrastructure Lead
* **Context:** Architectural specification for Roadmap Phase 9 (`7.36`–`7.39`), replacing flat intra-cluster networking and legacy `Ingress` annotations with default-deny L3/L4 microsegmentation, Envoy Gateway / Kubernetes Gateway API v1, embedded OWASP CRS WAF inspection, and an authenticated WebRTC STUN/TURN media relay.

---

## 1. Context and Problem Statement

1. **Flat Intra-Cluster Pod Network:** Without namespace-wide `NetworkPolicy` isolation, any compromised pod could attempt TCP connections directly to `postgres-core:5432`, sibling Domain APIs (`:50051–50054`), or external mocks.
2. **Legacy `networking.k8s.io/v1` `Ingress` Limitations:** Annotation-heavy NGINX Ingress objects lacked standard cross-namespace route governance, native `GRPCRoute` support, and declarative L7 `SecurityPolicy` CIDR enforcement.
3. **No Application-Layer WAF Inspection at the BFF Boundary:** BFF endpoints accepted JSON REST payloads without automated OWASP Core Rule Set (CRS) inspection for SQLi, XSS, LFI, and RCE probes.
4. **WebRTC NAT Traversal Dependency:** Customer habitat video streaming lacked an in-cluster STUN/TURN media relay issuing short-lived credentials.

---

## 2. Decision Outcome

1. **Default-Deny Zero-Trust `NetworkPolicy` Matrix (`7.36`):**
   * Deployed `default-deny-all` (`policyTypes: [Ingress, Egress]`, `podSelector: {}`) across `oci-core`, `oci-dev`, `oci-internal`, and `external-sim` (`k8s/*/network-policies.yaml`).
   * Enforced explicit allowlists guaranteeing BFFs cannot reach `:5432` or `:50051–50054`, Domain APIs have zero east-west paths and egress strictly to PostgreSQL `:5432`, and `external-sim` mocks have zero database egress (verified by Check #25 in `scripts/verify-ci-contracts.sh`).
2. **Kubernetes Gateway API v1 Migration (`7.37`):**
   * Removed all legacy `kind: Ingress` manifests and deployed Envoy Gateway (`v1.2.4`) with `GatewayClass/oci-gateway-class` and `Gateway/oci-edge-gateway` ([`k8s/ha/lan/tls/gateway.yaml`](../../k8s/ha/lan/tls/gateway.yaml)).
   * Defined namespace-scoped `HTTPRoute` and `GRPCRoute` resources (`k8s/*/http-routes.yaml`, `k8s/monitoring/gatus.yaml`, `k8s/ha/storage/longhorn-ui-ingress.yaml`) plus Envoy `SecurityPolicy` CIDR allowlists for internal admin UIs.
3. **Embedded Coraza OWASP CRS WAF Middleware (`7.38`):**
   * Implemented [`pkg/waf/coraza.go`](../../pkg/waf/coraza.go) inspecting URI, query args, headers, and bodies (1 MiB cap, bypassing SSE streams) on both `customer-bff` and `employee-bff`.
   * Blocks return `HTTP 403` (`WAF_RULE_VIOLATION`), record `oci.waf.blocked_requests.total`, and emit CWE-532-safe logs.
4. **In-Cluster `coturn` STUN/TURN Media Relay (`7.39`):**
   * Deployed `coturn` (`k8s/oci-core/coturn.yaml`, `k8s/oci-dev/coturn.yaml`) configured with `--use-auth-secret` on ports `3478` and `49152–49200`.
   * Added authenticated `GET /api/v1/customer/webrtc/ice-servers` on `customer-bff` returning 1-hour ephemeral HMAC-SHA1 TURN credentials to `web/customer-portal`.

---

## 3. Consequences

* **Layered Network & Application Defense:** Even if an application vulnerability occurred in a BFF pod, L3/L4 `NetworkPolicy` blocks direct database or Domain API access, while Coraza blocks common OWASP Top 10 payloads at the HTTP boundary.
* **Standardized Multi-Namespace Routing:** Gateway API enables explicit cross-namespace `HTTPRoute` ownership and declarative `SecurityPolicy` controls.
