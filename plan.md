1. **Add script `scripts/verify-deployed-images.sh`**
   - Create a bash script that iterates over given deployments (or all deployments in the namespace) and verifies that every container image has the expected `:<sha-tag>` tag.
   - Use `kubectl get deployment ... -o jsonpath='{..image}'`.
   - Exit 1 if any image tag mismatches.
2. **Modify `.forgejo/workflows/ci.yaml`**
   - Instead of just `kubectl apply -f k8s/oci-core/`, loop through the 13 deployments + mocks, set their image using `kubectl set image deployment/<app> <app>=localhost:32000/<app>:sha-${COMMIT_SHA}`, and then run `rollout status`.
   - Also add `kubectl set image deployment/<app> ... -n external-sim` for `mock-hardware`, `mock-bank`, `mock-broker` during production deployment.
   - Add calls to `scripts/verify-deployed-images.sh oci-dev sha-${COMMIT_SHA}` in `deploy-oci-dev`.
   - Add calls to `scripts/verify-deployed-images.sh oci-core sha-${COMMIT_SHA}` and `scripts/verify-deployed-images.sh external-sim sha-${COMMIT_SHA} mock-hardware mock-bank mock-broker` in `promote-oci-core`.
3. **Add mock Dockerfiles**
   - Since mock-bank and mock-broker deploy to the cluster, ensure their Dockerfiles exist (added `cmd/mock-bank/Dockerfile` and `cmd/mock-broker/Dockerfile`).
4. **Pre-commit step**
   - Ensure pre commit steps are executed
5. **Submit**
   - Push branch and create PR.
