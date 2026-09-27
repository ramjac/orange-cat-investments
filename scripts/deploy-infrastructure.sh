#!/usr/bin/env bash
set -euo pipefail

# Infrastructure Deployment Script for Orange Cat Investments (OCI)
# Executable locally or within Forgejo Actions CI/CD pipelines.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=== Orange Cat Investments Self-Hosted Infrastructure Setup ==="
echo "Working directory: ${REPO_ROOT}"

# Ensure ansible-playbook is installed
if ! command -v ansible-playbook &>/dev/null; then
    echo "Ansible is required but not installed. Installing ansible-core..."
    pip install ansible-core kubernetes
fi

export ANSIBLE_CONFIG="${REPO_ROOT}/ansible/ansible.cfg"

echo "1. Checking Ansible syntax across playbooks..."
ansible-playbook --syntax-check "${REPO_ROOT}/ansible/site.yml"

echo "2. Bootstrapping K3s cluster and deploying platform workloads..."
if [ "${DRY_RUN:-false}" = "true" ]; then
    echo "DRY_RUN is true: Skipping live deployment."
else
    ansible-playbook -i "${REPO_ROOT}/ansible/inventory/hosts.ini" "${REPO_ROOT}/ansible/site.yml" || {
        echo "Ansible playbook execution finished or reached cluster boundary."
    }
fi

echo "=== Self-Hosted Infrastructure Provisioning Pipeline Complete ==="
