#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# OpenBao Bootstrap & Dynamic PostgreSQL Database Engine Initializer (7.20c)
# ==============================================================================
# Idempotently initializes/unseals OpenBao in oci-core, enables Kubernetes auth,
# seeds KV-v2 platform secrets (preserving any existing live cluster values),
# and configures least-privilege schema-isolated PostgreSQL dynamic roles.
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ -z "${KUBECONFIG:-}" && -f "${REPO_ROOT}/thinker-kube.config" ]]; then
    export KUBECONFIG="${REPO_ROOT}/thinker-kube.config"
fi

DB_ONLY=false
if [[ "${1:-}" == "--db-only" ]]; then
    DB_ONLY=true
fi

bao_exec() {
    kubectl exec -i -n oci-core openbao-0 -- env BAO_ADDR="http://127.0.0.1:8200" "$@"
}

bao_authed() {
    kubectl exec -i -n oci-core openbao-0 -- env BAO_ADDR="http://127.0.0.1:8200" BAO_TOKEN="${ROOT_TOKEN}" "$@"
}

read_secret_key() {
    local ns="$1" secret="$2" key="$3" default_val="$4"
    local val=""
    val=$(kubectl get secret "${secret}" -n "${ns}" -o "jsonpath={.data.${key}}" 2>/dev/null | base64 -d 2>/dev/null || true)
    if [[ -n "${val}" ]]; then
        printf '%s' "${val}"
    else
        printf '%s' "${default_val}"
    fi
}

echo "Waiting for OpenBao pod (oci-core/openbao-0) to start..."
kubectl rollout status statefulset/openbao -n oci-core --timeout=120s 2>/dev/null || \
    kubectl wait --for=jsonpath='{.status.phase}'=Running pod/openbao-0 -n oci-core --timeout=120s

for _ in $(seq 1 30); do
    if kubectl exec -n oci-core openbao-0 -- sh -c 'BAO_ADDR=http://127.0.0.1:8200 bao status >/dev/null 2>&1 || [ $? -eq 2 ]'; then
        break
    fi
    sleep 2
done

STATUS_JSON=$(kubectl exec -n oci-core openbao-0 -- sh -c 'BAO_ADDR=http://127.0.0.1:8200 bao status -format=json || true')
INITIALIZED=$(python3 -c 'import json,sys; print(str(json.loads(sys.argv[1]).get("initialized", False)).lower())' "${STATUS_JSON}")

if [[ "${INITIALIZED}" != "true" ]]; then
    echo "Initializing OpenBao operator (single sandbox unseal share)..."
    INIT_JSON=$(bao_exec bao operator init -key-shares=1 -key-threshold=1 -format=json)
    UNSEAL_KEY=$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["unseal_keys_b64"][0])' "${INIT_JSON}")
    ROOT_TOKEN=$(python3 -c 'import json,sys; print(json.loads(sys.argv[1])["root_token"])' "${INIT_JSON}")

    kubectl create secret generic openbao-unseal-keys \
        --namespace oci-core \
        --from-literal=unseal_key="${UNSEAL_KEY}" \
        --from-literal=root_token="${ROOT_TOKEN}" \
        --dry-run=client -o yaml | kubectl apply -f - >/dev/null
    echo "Stored OpenBao unseal key and root token in cluster secret oci-core/openbao-unseal-keys."
else
    UNSEAL_KEY=$(read_secret_key "oci-core" "openbao-unseal-keys" "unseal_key" "")
    ROOT_TOKEN=$(read_secret_key "oci-core" "openbao-unseal-keys" "root_token" "")
    if [[ -z "${UNSEAL_KEY}" || -z "${ROOT_TOKEN}" ]]; then
        echo "ERROR: OpenBao is already initialized, but oci-core/openbao-unseal-keys is missing." >&2
        exit 1
    fi
fi

SEALED=$(python3 -c 'import json,sys; print(str(json.loads(sys.argv[1]).get("sealed", True)).lower())' "${STATUS_JSON}")
if [[ "${SEALED}" == "true" ]]; then
    echo "Unsealing OpenBao..."
    bao_exec bao operator unseal "${UNSEAL_KEY}" >/dev/null
fi

kubectl rollout status statefulset/openbao -n oci-core --timeout=90s

if [[ "${DB_ONLY}" != true ]]; then
    echo "Configuring OpenBao Kubernetes Auth & ESO policies..."
    bao_authed bao auth enable kubernetes 2>/dev/null || true
    bao_authed sh -c 'BAO_ADDR=http://127.0.0.1:8200 BAO_TOKEN="'"${ROOT_TOKEN}"'" bao write auth/kubernetes/config \
        kubernetes_host="https://kubernetes.default.svc"' >/dev/null

    bao_authed bao policy write eso-reader - <<'HCL' >/dev/null
path "secret/data/oci-*" {
  capabilities = ["read"]
}
path "secret/metadata/oci-*" {
  capabilities = ["read", "list"]
}
path "database/creds/*" {
  capabilities = ["read"]
}
HCL

    bao_authed bao write auth/kubernetes/role/eso-reader \
        bound_service_account_names="*" \
        bound_service_account_namespaces="oci-core,oci-dev,oci-internal,external-secrets" \
        policies="eso-reader" \
        ttl=1h >/dev/null

    echo "Enabling and seeding OpenBao KV-v2 engine (secret/)..."
    bao_authed bao secrets enable -path=secret kv-v2 2>/dev/null || true

    # Preserve existing live credentials when migrating existing clusters so running DBs/COTS remain aligned.
    CORE_PG_USER=$(read_secret_key "oci-core" "oci-core-secrets" "POSTGRES_USER" "oci_admin")
    CORE_PG_PASS=$(read_secret_key "oci-core" "oci-core-secrets" "POSTGRES_PASSWORD" "oci_secure_pass")
    CORE_PG_DB=$(read_secret_key "oci-core" "oci-core-secrets" "POSTGRES_DB" "oci")
    CORE_RMQ_USER=$(read_secret_key "oci-core" "oci-core-secrets" "RABBITMQ_DEFAULT_USER" "oci_rabbit")
    CORE_RMQ_PASS=$(read_secret_key "oci-core" "oci-core-secrets" "RABBITMQ_DEFAULT_PASS" "oci_rabbit_pass")
    CORE_MEILI_KEY=$(read_secret_key "oci-core" "oci-core-secrets" "MEILI_MASTER_KEY" "oci_meilisearch_master_key_32bytes")
    CORE_ZITADEL_KEY=$(read_secret_key "oci-core" "oci-core-secrets" "ZITADEL_MASTERKEY" "MasterKeyForZitadel32BytesLong!!")
    CORE_APCA_ID=$(read_secret_key "oci-core" "oci-core-secrets" "APCA_API_KEY_ID" "CK_OCI_TEST_KEY_01")
    CORE_APCA_SECRET=$(read_secret_key "oci-core" "oci-core-secrets" "APCA_API_SECRET_KEY" "SK_OCI_TEST_SECRET_01")
    CORE_PLAID_ID=$(read_secret_key "oci-core" "oci-core-secrets" "PLAID_CLIENT_ID" "mock-client-id")
    CORE_PLAID_SECRET=$(read_secret_key "oci-core" "oci-core-secrets" "PLAID_SECRET" "mock-secret")
    CORE_TURN_SECRET=$(read_secret_key "oci-core" "oci-core-secrets" "TURN_AUTH_SECRET" "")
    if [[ -z "${CORE_TURN_SECRET}" ]]; then
        CORE_TURN_SECRET=$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')
    fi
    INT_NC_USER=$(read_secret_key "oci-internal" "app-secrets" "NEXTCLOUD_ADMIN_USER" "admin")
    INT_NC_PASS=$(read_secret_key "oci-internal" "app-secrets" "NEXTCLOUD_ADMIN_PASSWORD" "oci_nextcloud_admin_pass")

    bao_authed bao kv put secret/oci-core/config \
        DATABASE_URL="postgres://${CORE_PG_USER}:${CORE_PG_PASS}@postgres-core-service.oci-core.svc.cluster.local:5432/${CORE_PG_DB}?sslmode=disable" \
        POSTGRES_USER="${CORE_PG_USER}" \
        POSTGRES_PASSWORD="${CORE_PG_PASS}" \
        POSTGRES_DB="${CORE_PG_DB}" \
        RABBITMQ_URL="amqp://${CORE_RMQ_USER}:${CORE_RMQ_PASS}@rabbitmq-service.oci-core.svc.cluster.local:5672/" \
        VALKEY_ADDR="valkey-service.oci-core.svc.cluster.local:6379" \
        RABBITMQ_DEFAULT_USER="${CORE_RMQ_USER}" \
        RABBITMQ_DEFAULT_PASS="${CORE_RMQ_PASS}" \
        MEILI_MASTER_KEY="${CORE_MEILI_KEY}" \
        ZITADEL_MASTERKEY="${CORE_ZITADEL_KEY}" \
        APCA_API_KEY_ID="${CORE_APCA_ID}" \
        APCA_API_SECRET_KEY="${CORE_APCA_SECRET}" \
        PLAID_CLIENT_ID="${CORE_PLAID_ID}" \
        PLAID_SECRET="${CORE_PLAID_SECRET}" \
        TURN_AUTH_SECRET="${CORE_TURN_SECRET}" \
        NEXTCLOUD_ADMIN_USER="${INT_NC_USER}" \
        NEXTCLOUD_ADMIN_PASS="${INT_NC_PASS}" >/dev/null

    DEV_PG_USER=$(read_secret_key "oci-dev" "oci-dev-secrets" "POSTGRES_USER" "oci_admin")
    DEV_PG_PASS=$(read_secret_key "oci-dev" "oci-dev-secrets" "POSTGRES_PASSWORD" "oci_secure_pass")
    DEV_PG_DB=$(read_secret_key "oci-dev" "oci-dev-secrets" "POSTGRES_DB" "oci_dev")
    DEV_RMQ_USER=$(read_secret_key "oci-dev" "oci-dev-secrets" "RABBITMQ_DEFAULT_USER" "oci_rabbit")
    DEV_RMQ_PASS=$(read_secret_key "oci-dev" "oci-dev-secrets" "RABBITMQ_DEFAULT_PASS" "oci_rabbit_pass")
    DEV_MEILI_KEY=$(read_secret_key "oci-dev" "oci-dev-secrets" "MEILI_MASTER_KEY" "oci_meilisearch_master_key_32bytes")
    DEV_ZITADEL_KEY=$(read_secret_key "oci-dev" "oci-dev-secrets" "ZITADEL_MASTERKEY" "MasterKeyForZitadel32BytesLong!!")
    DEV_TURN_SECRET=$(read_secret_key "oci-dev" "oci-dev-secrets" "TURN_AUTH_SECRET" "")
    if [[ -z "${DEV_TURN_SECRET}" ]]; then
        DEV_TURN_SECRET=$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')
    fi

    bao_authed bao kv put secret/oci-dev/config \
        DATABASE_URL="postgres://${DEV_PG_USER}:${DEV_PG_PASS}@postgres-dev-service.oci-dev.svc.cluster.local:5432/${DEV_PG_DB}?sslmode=disable" \
        POSTGRES_USER="${DEV_PG_USER}" \
        POSTGRES_PASSWORD="${DEV_PG_PASS}" \
        POSTGRES_DB="${DEV_PG_DB}" \
        RABBITMQ_URL="amqp://${DEV_RMQ_USER}:${DEV_RMQ_PASS}@rabbitmq-service.oci-dev.svc.cluster.local:5672/" \
        VALKEY_ADDR="valkey-service.oci-dev.svc.cluster.local:6379" \
        RABBITMQ_DEFAULT_USER="${DEV_RMQ_USER}" \
        RABBITMQ_DEFAULT_PASS="${DEV_RMQ_PASS}" \
        MEILI_MASTER_KEY="${DEV_MEILI_KEY}" \
        ZITADEL_MASTERKEY="${DEV_ZITADEL_KEY}" \
        APCA_API_KEY_ID="DEV_KEY_01" \
        APCA_API_SECRET_KEY="DEV_SECRET_01" \
        PLAID_CLIENT_ID="mock-dev-client" \
        PLAID_SECRET="mock-dev-secret" \
        TURN_AUTH_SECRET="${DEV_TURN_SECRET}" \
        NEXTCLOUD_ADMIN_USER="${INT_NC_USER}" \
        NEXTCLOUD_ADMIN_PASS="${INT_NC_PASS}" >/dev/null

    INT_PG_USER=$(read_secret_key "oci-internal" "app-secrets" "POSTGRES_USER" "oci_admin")
    INT_PG_PASS=$(read_secret_key "oci-internal" "app-secrets" "POSTGRES_PASSWORD" "oci_secure_pass")
    INT_PG_DB=$(read_secret_key "oci-internal" "app-secrets" "POSTGRES_DB" "oci_internal")
    INT_FG_PASS=$(read_secret_key "oci-internal" "app-secrets" "FORGEJO_DB_PASSWD" "${INT_PG_PASS}")
    INT_SNIPE_KEY=$(read_secret_key "oci-internal" "app-secrets" "SNIPEIT_APP_KEY" "base64:dGVzdGtleWZvcnNuaXBlaXRhcHBpbnByb2R1Y3Rpb24=")
    INT_HARBOR_PASS=$(read_secret_key "oci-internal" "app-secrets" "HARBOR_ADMIN_PASSWORD" "OciHarborAdmin123!")
    INT_HARBOR_CORE_SECRET=$(read_secret_key "oci-internal" "app-secrets" "HARBOR_CORE_SECRET" "oci_harbor_core_secret_32bytes!!")

    bao_authed bao kv put secret/oci-internal/config \
        POSTGRES_USER="${INT_PG_USER}" \
        POSTGRES_PASSWORD="${INT_PG_PASS}" \
        POSTGRES_DB="${INT_PG_DB}" \
        NEXTCLOUD_ADMIN_USER="${INT_NC_USER}" \
        NEXTCLOUD_ADMIN_PASSWORD="${INT_NC_PASS}" \
        FORGEJO_DB_PASSWD="${INT_FG_PASS}" \
        SNIPEIT_APP_KEY="${INT_SNIPE_KEY}" \
        HARBOR_ADMIN_PASSWORD="${INT_HARBOR_PASS}" \
        HARBOR_CORE_SECRET="${INT_HARBOR_CORE_SECRET}" >/dev/null

    # Seed Cosign signing keypair in OpenBao (preserving existing key if present) and sync ConfigMap/oci-cosign-pub
    EXISTING_COSIGN_PUB=$(bao_authed bao kv get -field=COSIGN_PUB secret/oci-core/cosign 2>/dev/null || true)
    if [[ -z "${EXISTING_COSIGN_PUB}" ]] && command -v openssl &>/dev/null; then
        COSIGN_TMP=$(mktemp -d)
        openssl ecparam -name prime256v1 -genkey -noout -out "${COSIGN_TMP}/ec.key" 2>/dev/null
        openssl pkcs8 -topk8 -nocrypt -in "${COSIGN_TMP}/ec.key" -out "${COSIGN_TMP}/cosign.key" 2>/dev/null
        openssl ec -in "${COSIGN_TMP}/ec.key" -pubout -out "${COSIGN_TMP}/cosign.pub" 2>/dev/null
        COSIGN_PRIV_VAL=$(cat "${COSIGN_TMP}/cosign.key")
        COSIGN_PUB_VAL=$(cat "${COSIGN_TMP}/cosign.pub")
        rm -rf "${COSIGN_TMP}"
        bao_authed bao kv put secret/oci-core/cosign \
            COSIGN_KEY="${COSIGN_PRIV_VAL}" \
            COSIGN_PUB="${COSIGN_PUB_VAL}" >/dev/null
        EXISTING_COSIGN_PUB="${COSIGN_PUB_VAL}"
    fi
    if [[ -n "${EXISTING_COSIGN_PUB}" ]]; then
        for cosign_ns in oci-core oci-dev external-sim; do
            kubectl create configmap oci-cosign-pub \
                --namespace "${cosign_ns}" \
                --from-literal=cosign.pub="${EXISTING_COSIGN_PUB}" \
                --dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1 || true
        done
    fi
    echo "OpenBao KV-v2 paths seeded."
fi

configure_pg_engine_for_env() {
    local ns="$1" sts="$2" svc="$3" db="$4" suffix="$5" kv_path="$6"
    if ! kubectl get pod "${sts}-0" -n "${ns}" >/dev/null 2>&1; then
        return 0
    fi
    local ready
    ready=$(kubectl get pod "${sts}-0" -n "${ns}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "False")
    if [[ "${ready}" != "True" ]]; then
        echo "Skipping OpenBao DB role setup for ${ns}/${sts} (not yet Ready)."
        return 0
    fi

    echo "Configuring OpenBao Database Secrets Engine for ${ns}/${sts} (${db})..."
    local pg_user pg_pass conn_name="postgres-${suffix}"
    pg_user=$(bao_authed bao kv get -field=POSTGRES_USER "${kv_path}")
    pg_pass=$(bao_authed bao kv get -field=POSTGRES_PASSWORD "${kv_path}")

    bao_authed bao secrets enable database 2>/dev/null || true
    bao_authed bao write "database/config/${conn_name}" \
        plugin_name=postgresql-database-plugin \
        allowed_roles="workforce-${suffix},facilities-${suffix},core-invest-${suffix},ops-${suffix}" \
        connection_url="postgresql://{{username}}:{{password}}@${svc}.${ns}.svc.cluster.local:5432/${db}?sslmode=disable" \
        username="${pg_user}" \
        password="${pg_pass}" >/dev/null

    local domains=("workforce:workforce" "facilities:facilities" "core-invest:core_invest" "ops:ops")
    for entry in "${domains[@]}"; do
        local role_prefix="${entry%%:*}"
        local schema="${entry##*:}"
        local revoke_stmts=""
        for other_schema in workforce facilities core_invest ops; do
            if [[ "${other_schema}" != "${schema}" ]]; then
                revoke_stmts+="REVOKE ALL PRIVILEGES ON SCHEMA ${other_schema} FROM \"{{name}}\";"
            fi
        done
        local creation_sql="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; ${revoke_stmts} GRANT USAGE ON SCHEMA ${schema} TO \"{{name}}\"; GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA ${schema} TO \"{{name}}\"; GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA ${schema} TO \"{{name}}\"; ALTER DEFAULT PRIVILEGES IN SCHEMA ${schema} GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO \"{{name}}\"; ALTER DEFAULT PRIVILEGES IN SCHEMA ${schema} GRANT USAGE, SELECT ON SEQUENCES TO \"{{name}}\";"
        local revocation_sql="SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE usename = '{{name}}'; DROP ROLE IF EXISTS \"{{name}}\";"

        bao_authed bao write "database/roles/${role_prefix}-${suffix}" \
            db_name="${conn_name}" \
            creation_statements="${creation_sql}" \
            revocation_statements="${revocation_sql}" \
            default_ttl="24h" \
            max_ttl="48h" >/dev/null
    done
}

configure_pg_engine_for_env "oci-core" "postgres-core" "postgres-core-service" "oci" "core" "secret/oci-core/config"
configure_pg_engine_for_env "oci-dev" "postgres-dev" "postgres-dev-service" "oci_dev" "dev" "secret/oci-dev/config"

# Remove legacy unmanaged static Secrets so ExternalSecret (creationPolicy: Owner) adopts them cleanly.
for pair in "oci-core:oci-core-secrets" "oci-dev:oci-dev-secrets" "oci-internal:app-secrets"; do
    ns="${pair%%:*}"
    sec="${pair##*:}"
    if kubectl get secret "${sec}" -n "${ns}" >/dev/null 2>&1; then
        owner_kind=$(kubectl get secret "${sec}" -n "${ns}" -o jsonpath='{.metadata.ownerReferences[0].kind}' 2>/dev/null || true)
        if [[ "${owner_kind}" != "ExternalSecret" ]]; then
            echo "Removing legacy static Secret ${ns}/${sec} so ESO ExternalSecret manages it..."
            kubectl delete secret "${sec}" -n "${ns}" --ignore-not-found >/dev/null 2>&1 || true
        fi
    fi
done

echo "OpenBao initialization and secret configuration complete."
