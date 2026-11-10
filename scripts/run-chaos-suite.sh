#!/usr/bin/env bash
set -euo pipefail

# Define a safe kubectl wrapper
safe_kubectl() {
  local args=("$@")
  local ns=""
  local resource=""
  local skip_next=false

  # Extract namespace and resource
  for (( i=0; i<${#args[@]}; i++ )); do
    if $skip_next; then
        skip_next=false
        continue
    fi
    if [[ "${args[i]}" == "-n" || "${args[i]}" == "--namespace" ]]; then
      ns="${args[$((i+1))]}"
      skip_next=true
      continue
    fi
    if [[ "${args[i]}" == "-f" ]]; then
        resource="${args[$((i+1))]}"
        skip_next=true
        continue
    fi
    if [[ "${args[i]}" != "-"* && "${args[i]}" != "apply" && "${args[i]}" != "delete" && "${args[i]}" != "scale" && "${args[i]}" != "get" && "${args[i]}" != "networkpolicy" ]]; then
        if [[ "$resource" == "" ]]; then
            resource="${args[i]}"
        else
            resource="$resource ${args[i]}"
        fi
    fi
  done

  if [[ "${args[0]}" == "apply" || "${args[0]}" == "delete" || "${args[0]}" == "scale" ]]; then
      if [[ "$ns" != "oci-dev" ]]; then
        if [[ "$ns" == "external-sim" ]]; then
           if [[ "$resource" != *"-dev"* ]]; then
             echo "Error: Refusing to mutate external-sim resource without -dev suffix: $resource"
             return 1
           fi
        else
            echo "Error: Refusing to mutate namespace $ns"
            return 1
        fi
      fi
  fi

  kubectl "${args[@]}"
}

cleanup() {
  echo "Cleaning up chaos policies and restoring replica counts..."
  kubectl delete -f k8s/chaos/oci-dev-deny-internal-egress.yaml --ignore-not-found -n oci-dev || true
  kubectl delete networkpolicy deny-bff-bll --ignore-not-found -n oci-dev || true
  kubectl delete networkpolicy deny-bll-domain --ignore-not-found -n oci-dev || true
  kubectl scale statefulset postgres-dev --replicas=1 -n oci-dev || true
  # Reset mock faults
  curl -s -X DELETE http://mock-bank-dev-service.external-sim.svc.cluster.local:8088/sim/faults || true
  curl -s -X DELETE http://mock-broker-dev-service.external-sim.svc.cluster.local:8089/sim/faults || true
}
trap cleanup EXIT INT ERR


# 1. COTS outage
echo "Scenario: COTS outage"
safe_kubectl apply -f k8s/chaos/oci-dev-deny-internal-egress.yaml -n oci-dev
sleep 5
safe_kubectl delete -f k8s/chaos/oci-dev-deny-internal-egress.yaml -n oci-dev

# 2. Tier partitions
echo "Scenario: Tier partitions"
cat << 'POLICY' | safe_kubectl apply -f - -n oci-dev
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-bff-bll
  namespace: oci-dev
spec:
  podSelector:
    matchLabels:
      tier: bll
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchExpressions:
        - key: tier
          operator: NotIn
          values:
          - bff
POLICY
sleep 5
safe_kubectl delete networkpolicy deny-bff-bll -n oci-dev

cat << 'POLICY' | safe_kubectl apply -f - -n oci-dev
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-bll-domain
  namespace: oci-dev
spec:
  podSelector:
    matchLabels:
      tier: domain-api
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchExpressions:
        - key: tier
          operator: NotIn
          values:
          - bll
POLICY
sleep 5
safe_kubectl delete networkpolicy deny-bll-domain -n oci-dev

# 3. Database restart
echo "Scenario: Database restart"
safe_kubectl scale statefulset postgres-dev --replicas=0 -n oci-dev
sleep 5
safe_kubectl scale statefulset postgres-dev --replicas=1 -n oci-dev
# wait for postgres to be ready
sleep 15

# 4. Pod kills
echo "Scenario: Pod kills"
safe_kubectl get pod -l tier=domain-api --field-selector=status.phase=Running -n oci-dev --no-headers | head -n 1 | awk '{print $1}' | xargs -I {} safe_kubectl delete pod {} -n oci-dev || true
safe_kubectl get pod -l tier=bll --field-selector=status.phase=Running -n oci-dev --no-headers | head -n 1 | awk '{print $1}' | xargs -I {} safe_kubectl delete pod {} -n oci-dev || true
safe_kubectl get pod -l tier=bff --field-selector=status.phase=Running -n oci-dev --no-headers | head -n 1 | awk '{print $1}' | xargs -I {} safe_kubectl delete pod {} -n oci-dev || true
sleep 15

# 5. External outage
echo "Scenario: External outage"
# Only execute if simulator handles faults
go run ./simulation/cmd/simulator -fault-scenarios -bff-url "http://customer-bff-service.oci-dev.svc.cluster.local:8080" || true

# Simulator pass
echo "Running simulator"
go run ./simulation/cmd/simulator -bff-url "http://customer-bff-service.oci-dev.svc.cluster.local:8080" -validate-flags
