#!/usr/bin/env bash
set -euo pipefail

echo "=============================================================="
echo "  Deploying Latest OCI Platform                               "
echo "=============================================================="

# 1. Run unit tests locally before pushing
echo "Running local tests..."
go test -race ./...

# 2. Sync to Forgejo - This pushes code to the cluster's Git service,
# which will then automatically trigger the CI pipeline to build images
# and deploy to the oci-dev environment.
echo "Syncing repository to Forgejo (triggers CI pipeline)..."
./scripts/sync-repo-to-forgejo.sh

# 3. Wait for the CI pipeline to finish building and deploying
echo "Waiting for Forgejo Actions pipeline..."
./scripts/ci-status.sh --wait

# 4. Verify that the new image SHA was correctly rolled out to oci-dev
echo "Verifying deployed images in oci-dev..."
# The sync script pushes with a commit SHA, and the CI uses this SHA tag.
# We fetch the current local commit SHA to check.
COMMIT_SHA=$(git rev-parse --short HEAD)
./scripts/verify-deployed-images.sh oci-dev "sha-${COMMIT_SHA}"

# 5. Run the simulator against the dev environment to verify functionality
echo "Running simulation against oci-dev..."
# We run the simulator workstation client against the BFFs on localhost.
# Assuming standard port-forwarding is active for oci-dev, or using dev domains.
go run ./simulation/cmd/simulator -mode step -iterations 1

echo "=============================================================="
echo "  Successfully Built, Deployed, and Verified Latest to DEV!   "
echo "=============================================================="
