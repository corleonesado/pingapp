# Postmortem

Real incidents during the build, in chronological order. Every one of these forced a real fix — none were resolved by disabling the check that caught them. They are documented here because the *why* matters as much as the eventual code.

## 1. Trivy gate caught Go stdlib CVEs in the v0.1.0 builder

**Symptom.** First CI build of the full pipeline went red on `build & scan`. Trivy reported HIGH/CRITICAL CVEs against `app/server` (the compiled Go binary), specifically CVE-2026-32283 / 33811 / 33814 / 39820 / 39836 / 42499 in `crypto/tls`, `net/http`, `net/url`, and friends. Fixed versions: Go ≥ 1.25.10 / 1.26.3.

**Why it happened.** The Dockerfile builder image was `golang:1.22-alpine`. The compiled binary therefore carried a 1.22 stdlib. The CVEs were patched in newer toolchains, so the image inherited a vulnerability set that had been fixed upstream months ago.

**Fix.** Bumped `ARG GO_VERSION=1.22` → `ARG GO_VERSION=1.26` in the Dockerfile so the multi-stage build uses `golang:1.26-alpine`. Kept `go.mod`'s `go` directive at 1.22 (language floor); the bump is *toolchain*, not language. Re-scan → 0 HIGH/CRITICAL.

**Why not bypass.** The brief explicitly calls out "Turning off or removing the image scan step in the pipeline" as a thing to avoid. `.trivyignore` for fixable CVEs would have been a worse signal than the actual one-line fix.

**Captured in.** [ADR-0004](decisions/0004-image-scanning-gate.md), `Dockerfile`, `CHANGELOG.md`.

## 2. `golangci-lint-action@v6` doesn't support golangci-lint v2

**Symptom.** First PR's `lint` job failed in ~5 seconds with `invalid version string 'v2.12.2', golangci-lint v2 is not supported by golangci-lint-action v6, you must update to golangci-lint-action v7`.

**Why it happened.** I'd written `.golangci.yml` in v2 format (correct — that's current) and used `golangci/golangci-lint-action@v6` in the workflow (one major version behind what v2 needs).

**Fix.** Bumped the action to `@v7`.

**Lesson.** Pin actions to a tag, but track major-version changes when bumping the underlying linter — they're released together for a reason. Added `actionlint` as a CI job after this incident so similar workflow-validity bugs are caught locally.

## 3. `gitleaks-action@v2` got 403 on PR commits API

**Symptom.** First PR's `gitleaks` job failed with `RequestError [HttpError]: Resource not accessible by integration` on `GET /repos/.../pulls/1/commits`.

**Why it happened.** `gitleaks-action` uses the GitHub API to list PR commits, which needs `pull-requests: read` permission. My workflow's top-level `permissions: contents: read` didn't grant that, and tightening the permission boundary was an explicit design choice (principle of least privilege).

**Fix.** Switched to running the `gitleaks` binary directly:

```yaml
- name: Install gitleaks
  run: |
    curl -sSfL "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz" \
      | tar -xz gitleaks
- name: Scan git history for secrets
  run: ./gitleaks detect --source . --redact --no-banner
```

This bypasses the GitHub API entirely — gitleaks reads the cloned repo's git history. Deterministic, no permission grants, no license check.

**Lesson.** Third-party Actions can require permissions you don't want to grant. Falling back to a vendored binary is often simpler than widening least-privilege.

## 4. `aquasecurity/trivy-action@0.28.0` doesn't exist

**Symptom.** `build & scan` failed at action resolution: `Unable to resolve action aquasecurity/trivy-action@0.28.0, unable to find version 0.28.0`.

**Why it happened.** Guessed the version from memory instead of looking it up.

**Fix.** `gh release list -R aquasecurity/trivy-action` → latest is `v0.36.0`. Pinned to that.

**Lesson.** Action tags don't follow a single convention (some use the `v` prefix, some don't; some skip versions). Always verify against the actual release list.

## 5. ArgoCD CRDs failed `kubectl apply` with "annotations Too long"

**Symptom.** `make argocd-install` failed partway through with `The CustomResourceDefinition "applicationsets.argoproj.io" is invalid: metadata.annotations: Too long: may not be more than 262144 bytes`.

**Why it happened.** `kubectl apply` stores the last-applied state in a `kubectl.kubernetes.io/last-applied-configuration` annotation. ArgoCD's `ApplicationSet` CRD is large enough that this annotation exceeds Kubernetes' 256 KB limit.

**Fix.** Switched the Makefile target to use server-side apply:

```bash
kubectl apply -n argocd -f .../install.yaml --server-side --force-conflicts
```

Server-side apply tracks field ownership via the API server instead of stuffing a last-applied snapshot into an annotation. Same result, no size limit.

**Lesson.** When upstream manifests are too large for client-side apply, server-side apply is the documented escape hatch. Documented in the RUNBOOK so the next person doesn't have to discover it.

## 6. `helm uninstall` deleted resources ArgoCD was managing

**Symptom.** During ArgoCD bring-up, ran `helm uninstall pingapp` to remove a stale Helm release. The pods, service, and ingress all terminated. ArgoCD's Application went `OutOfSync` momentarily, then recreated everything from the chart.

**Why it happened.** ArgoCD's `selfHeal: true` does exactly what it says — observed live state ≠ desired state ⇒ reconcile. The Helm uninstall blew away the resources; ArgoCD put them back within the next sync window.

**Outcome.** Self-healing worked correctly. The cluster ended up in the desired state. Slight downtime during the gap.

**Lesson.** With ArgoCD owning a deployment, `helm uninstall` and `kubectl delete` are not the same operation any more — they're temporary disturbances. To actually remove the app you have to delete the Application first. The RUNBOOK's rollback section was rewritten to make this explicit.

## 7. Credential leaked in chat during setup

**Symptom.** During the ArgoCD repo-credential setup, a GitHub fine-grained PAT was pasted in a chat session (intended as a test of the procedure, not a real credential, but it had real format).

**Mitigation.** Treated as compromised the moment it left its intended secret store. Token revoked the same minute. New PAT generated, written into the `pingapp-repo` Kubernetes Secret via the `stty -echo` + heredoc procedure that hides the value at every step (terminal echo off, no shell history line, no chat surface).

**Lesson.** Wrote the secret-leak protocol into `SECURITY.md` and the rotation procedure into `RUNBOOK.md`. The chat-only path is the one that's hardest to recover from because the value is in someone else's transcript.

## 8. cardinality risk in `http_requests_total{path}` label

**Symptom.** None — caught during design review of the metrics middleware.

**Risk.** If `path` were derived from `r.URL.Path` directly, a 404 scanner hitting `/admin`, `/.env`, `/wp-login.php` etc. would mint a new label permutation per request. With `request_duration_seconds` histograms, that quickly exhausts cardinality budgets in Prometheus.

**Fix.** Defined `routeOf(*http.Request) string` in `cmd/server/main.go` that allow-lists `{ /ping, /healthz, /version, /chaos }` and collapses everything else to `"unknown"`. `/metrics` itself is registered outside the metrics middleware so scrapes don't instrument themselves either.

**Lesson.** Default behaviour of HTTP instrumentation libraries is to use the raw path as a label. That's the wrong default for any service that takes traffic from outside the cluster. The fix is one switch statement and two lines of comment.
