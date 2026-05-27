# Roadmap

4-day plan for the Insider One DevOps case study, **Track B** (local minikube + tunnel).

Update this file as you go. It's both the plan and the progress tracker.

**Legend**: `[ ]` not started · `[~]` in progress · `[x]` done · `[-]` skipped (with reason)

---

## Day 1 — Foundation

> **Deliverable**: A locally-running service via `docker run`, in a clean GitHub repo.

### 1.1 Tiny HTTP service
- [x] `go mod init github.com/corleonesado/pingapp`
- [x] `cmd/server/main.go` with HTTP server + graceful shutdown
- [x] `GET /ping` → `pong` (text/plain, 200)
- [x] `GET /healthz` → `200 OK` (used by K8s probes)
- [x] `GET /version` → `{"version":"<sha>"}` (via `-ldflags "-X main.version=..."`)
- [x] Env-driven config: `PORT`, `LOG_LEVEL`
- [x] Structured JSON logger (`log/slog`)
- [x] `request_id` middleware (UUID v4 per request, propagated in logs)
- [x] At least 2 unit tests (`internal/handlers/handlers_test.go`) — 5 tests, 90.6% coverage

### 1.2 Containerize
- [x] Multi-stage `Dockerfile`:
  - builder: `golang:1.22-alpine` → `go build -ldflags "-s -w -X main.version=$SHA"`
  - runtime: `gcr.io/distroless/static:nonroot`
- [x] `USER 65532:65532` (distroless `nonroot`)
- [x] `.dockerignore` (exclude tests, docs, .git, .github)
- [x] `docker-compose.yaml` for local dev (port 8080, env vars)
- [x] Verify: `docker build -t pingapp:dev . && docker run -p 8080:8080 pingapp:dev` — image 14MB
- [x] `curl localhost:8080/ping` → `pong`

### 1.3 Repo hygiene
- [x] `README.md` skeleton (overview, setup, run, env, architecture)
- [x] `.gitignore` (Go + IDE + `.env`)
- [x] `.env.example` documents every env var
- [x] `CODEOWNERS` (just you for now)
- [x] `.github/PULL_REQUEST_TEMPLATE.md` (what, why, screenshots, checklist)
- [ ] Branch protection on `main`: require PR, require CI green (Day 3, after CI exists)
- [x] First 2-3 conventional commits — 3 commits on `main` (`f777651`, `4b2c5a7`, `c05d661`)

### 1.4 Decisions to capture
- [x] ADR-0001: language and runtime (Go + distroless, covers base image rationale)

### Day 1 checkpoint
- [x] `docker build` and `docker run` succeed cleanly
- [x] `curl localhost:8080/ping` returns `pong`
- [x] `curl localhost:8080/healthz` returns 200
- [x] `curl localhost:8080/version` returns the SHA — `{"version":"c05d661"}`
- [x] `git log` is clean (3 conventional commits, no secrets)
- [ ] `gitleaks detect --no-git` passes locally — defer to Day 3 (wired into CI)

---

## Day 2 — Kubernetes & Helm

> **Deliverable**: `helm upgrade --install` works against minikube with visible dev vs prod differences, rollback proven.

### 2.1 Helm chart
- [x] Hand-rolled chart (not from `helm create`) — rationale in ADR-0002
- [x] Templates: `deployment.yaml`, `service.yaml`, `ingress.yaml`, `configmap.yaml`, `pdb.yaml`, `_helpers.tpl`
- [x] `Chart.yaml` with `appVersion` aligned to image tag
- [x] `image.repository` and `image.tag` parameterized
- [x] `helm lint charts/pingapp` passes (default + dev + prod)

### 2.2 Environments
- [x] `values-dev.yaml`: 1 replica, low resources, host `dev.pingapp.local`, log level `debug`
- [x] `values-prod.yaml`: 2 replicas, higher resources, host `pingapp.local`, PDB enabled
- [x] `/etc/hosts` entries documented in README for local ingress
- [x] Diff documented in README (table form)

### 2.3 Probes & resources
- [x] `livenessProbe.httpGet.path: /healthz`, `initialDelaySeconds: 5`, `periodSeconds: 10`
- [x] `readinessProbe.httpGet.path: /healthz`, `periodSeconds: 5`
- [x] Optional `startupProbe` available via `probes.startup.enabled` (off by default)
- [x] `resources.requests`: CPU 50m / mem 32Mi (default) — 25m/16Mi in dev
- [x] `resources.limits`: CPU 200m / mem 64Mi (default) — 100m/32Mi in dev
- [x] Reasoning documented in README

### 2.4 Rollout & rollback
- [x] Demonstrated dev → prod upgrade (rev 2) — replicas 1→2, host change visible
- [x] Captured `kubectl rollout status deployment/pingapp` output
- [x] `helm rollback pingapp 1` succeeded (rev 3 = "Rollback to 1")
- [x] Captured `helm history pingapp`
- [x] Evidence saved to `docs/screenshots/day2-{01-dev-deploy,02-prod-rollout,03-rollback}.txt`

### 2.5 Bonus
- [x] PodDisruptionBudget — `policy/v1`, `minAvailable: 1`, enabled in prod only

### 2.6 Decisions to capture
- [x] ADR-0002: Helm over raw manifests / Kustomize

### Day 2 checkpoint
- [x] `make deploy-dev` → pods Running, probes Healthy
- [x] Prod values visibly differ (replicas, host, log level, PDB)
- [x] Rollback demonstrated with `helm history` evidence
- [x] Ingress reachable via `curl --resolve dev.pingapp.local:80:<minikube-ip> http://dev.pingapp.local/ping`

---

## Day 3 — CI/CD & supply-chain security

> **Deliverable**: Green CI run, image on GHCR, `v0.1.0` GitHub Release, auto-deploy to minikube on merge to main.

### 3.1 CI pipeline (`.github/workflows/ci.yml`)
- [x] Trigger: PRs to `main`, pushes to `main`
- [x] Job: `test` — `go test ./... -race -cover`
- [x] Job: `lint` — `golangci-lint` (`.golangci.yml`, v2)
- [x] Job: `build` — docker buildx with gha cache, tag `:<sha>`
- [x] Job: `scan` — Trivy on the built image, fail on HIGH/CRITICAL
- [x] Job: `push` (main only) — push to `ghcr.io/corleonesado/pingapp:<sha>`
- [x] Cache for Go modules (setup-go) and Docker layers (`type=gha`)
- [x] Concurrency group to cancel stale runs

### 3.2 Secrets & auth
- [x] GHCR push uses `${{ secrets.GITHUB_TOKEN }}` with `packages: write` permission
- [x] No PATs anywhere (GHCR via GITHUB_TOKEN; ArgoCD repo cred is a read-only credential, documented)
- [x] `gitleaks/gitleaks-action` step in CI
- [x] Verified locally: `gitleaks detect` → no leaks across all commits

### 3.3 Release hygiene
- [x] `CHANGELOG.md` in Keep a Changelog format with `[Unreleased]` + `[0.1.0]`
- [x] `.github/workflows/release.yml`: trigger on `v*.*.*` tag push
- [x] Release workflow: build, scan, push image with `:v0.1.0` and `:<sha>`, SBOM
- [x] Create GitHub Release with notes from CHANGELOG
- [ ] Cut `v0.1.0` (run `git tag v0.1.0 && git push origin v0.1.0` after CI is green)

### 3.4 Auto-deploy on merge
- [x] **Decision**: ArgoCD GitOps — ADR-0003
- [x] ArgoCD `Application` + bootstrap docs under `deploy/argocd/`, Makefile targets
- [x] ArgoCD installed on minikube (server-side apply for large CRD annotations)
- [x] Private-repo read cred provided as a labelled `Secret` in the `argocd` namespace
- [x] Application applied; initial sync `Synced + Healthy` against `ghcr.io/corleonesado/pingapp:v0.1.0`
- [x] **GitOps loop verified**: chart change pushed to `main` → ArgoCD synced and rolled out (`replicaCount` 2 → 3) in **~20s**, evidence in `docs/screenshots/day3-02-argocd-gitops-rollout.txt`

### 3.5 Bonus — Syft SBOM
- [x] SPDX SBOM generated by Syft in `release.yml`, attached to the GitHub Release

### 3.6 Decisions to capture
- [x] ADR-0003: auto-deploy strategy (ArgoCD GitOps)
- [x] ADR-0004: image scanning gate (Trivy thresholds)

### Day 3 checkpoint
- [x] Test PR → CI green end-to-end (PR #1, 4/4 jobs green)
- [x] Image at `ghcr.io/corleonesado/pingapp` (both `:<sha>` from main CI and `:v0.1.0` from release)
- [x] `v0.1.0` GitHub Release with extracted CHANGELOG notes + SPDX SBOM attached
- [x] Chart change pushed to `main` → ArgoCD synced and rolled out in ~20s
- [x] gitleaks + Trivy gates green (in CI and locally)

---

## Day 4 — Observability & docs

> **Deliverable**: Grafana dashboard + alert, public URL, RUNBOOK, SECURITY, ADRs, architecture diagram.

### 4.1 Logs & metrics (app side)
- [x] Structured JSON logs from Day 1 still emit (verified via `kubectl logs`)
- [x] `/metrics` endpoint using `prometheus/client_golang` (`internal/metrics`)
- [x] Counter `http_requests_total{method,path,status}` (route allow-list to bound cardinality)
- [x] Histogram `http_request_duration_seconds{method,path}` (default buckets)
- [x] Gauge `http_requests_in_flight`
- [x] Plus Go runtime + process collectors (`collectors.NewGoCollector`, `NewProcessCollector`)
- [x] 5 unit tests on the metrics middleware (85.7% coverage)

### 4.2 Prometheus + Grafana
- [x] `helm repo add prometheus-community ...`
- [x] `helm install kps prometheus-community/kube-prometheus-stack -n monitoring`
- [x] `ServiceMonitor` in our chart (Prometheus has 3/3 healthy pingapp targets)
- [x] Grafana dashboard JSON committed at `docs/grafana-dashboard.json` (RPS by status, latency p50/p95/p99, 5xx error rate, in-flight, pod restarts, active alerts)
- [x] `PrometheusRule` `PingappHighErrorRate`: `5xx rate > 5% for 2m`
- [x] Alert verified end-to-end: fired at 51.6% 5xx ratio against `/chaos`, resolved 150s after load stopped
- [x] Evidence in `docs/screenshots/day4-{01-alert-firing,02-alert-resolved}.txt`

### 4.3 Reproducibility (Track B IaC substitute)
- [x] `Makefile` covers every workflow command from CLAUDE.md (run/test/lint, docker, minikube, helm, deploy-dev/prod, rollback, history, status, argocd-*, obs-install, grafana, tunnel, bootstrap)
- [x] `scripts/bootstrap.sh` — checks docker / kubectl / helm / minikube / go / git / gh / cloudflared / golangci-lint / trivy / gitleaks; reports docker + minikube + helm-repo runtime state; idempotent
- [x] README documents: fresh-laptop story via `make bootstrap`

### 4.4 Architecture & docs
- [ ] `docs/architecture.png` (Excalidraw) — user to draw and export
- [x] `RUNBOOK.md` — one-page incident guide (restart, logs, rollback, PAT rotation, common failures, tunnel)
- [x] `SECURITY.md` — threat model, image hardening, supply chain, secret handling, known limits
- [x] **5 ADRs** in `docs/decisions/` (0001 language/runtime, 0002 Helm, 0003 ArgoCD, 0004 Trivy gate, 0005 cloudflared)

### 4.5 Public URL
- [x] `cloudflared tunnel --url http://$(minikube ip):80 --http-host-header pingapp.local` wired as `make tunnel`
- [x] Demoed end-to-end: `pong`, `{"version":"v0.1.1"}`, `HTTP/2 200` over TLS
- [x] Documented as **ephemeral** in README + ADR-0005 + `docs/screenshots/day4-03-cloudflared-tunnel.txt`
- [ ] Bonus (skipped for scope): named tunnel with a stable subdomain

### Day 4 checkpoint
- [x] Grafana dashboard JSON renders 6 panels including ALERTS
- [x] PrometheusRule fires and resolves on `/chaos` load (evidence captured)
- [x] RUNBOOK reads as a one-page incident guide
- [x] ADRs cover why Helm (0002), why this base image (0001), why this tunnel (0005), plus auto-deploy (0003) and Trivy gate (0004)
- [x] Public URL reachable from outside the network (Cloudflare edge → laptop)
- [x] Repo is fresh-laptop reproducible via `make bootstrap`

---

## Bonus track (only if Day 4 wraps early)

Pick **one** and do it well. Document in README what you learned.

- [ ] Policy-as-Code: Kyverno cluster policy denying `:latest` + root containers
- [ ] Supply chain: cosign sign + Syft SBOM in release workflow
- [ ] GitOps proper: ArgoCD `ApplicationSet` for dev + prod
- [ ] Chaos test: `kubectl delete pod` loop, screenshot recovery in Grafana
- [ ] Custom metric → Slack: domain counter → Alertmanager → Slack webhook
- [ ] Custom domain + TLS: cert-manager + Let's Encrypt

---

## Submission checklist (from the brief)

- [x] Private GitHub repo `corleonesado/pingapp` (invite `insider-one-devops` at submission time, or share link directly)
- [x] `README.md` with setup, run, env vars, architecture notes, Track B chosen
- [ ] Architecture diagram in repo — Excalidraw export to `docs/architecture.png` (user to draw)
- [x] Helm chart at `charts/pingapp/` with `values-dev.yaml` and `values-prod.yaml`
- [x] `.github/workflows/ci.yml` + `release.yml` with green runs (PR #1, PR #2, v0.1.0 + v0.1.1 releases)
- [x] Text-log evidence in `docs/screenshots/`: kubectl get pods, helm history, kubectl rollout status, ArgoCD sync, alert fire+resolve, cloudflared tunnel
- [x] Grafana dashboard JSON at `docs/grafana-dashboard.json`; alert demonstrated firing then resolving
- [x] Public URL via `make tunnel` (ephemeral; capture at submission time)
- [x] `RUNBOOK.md` and `SECURITY.md`
- [x] **5 ADRs** at `docs/decisions/0001..0005-*.md`

---

## Time budget

| Day | Hours (target) | Hours (actual) | Notes |
|---|---|---|---|
| 1 | 6 | | |
| 2 | 6 | | |
| 3 | 7 | | CI debugging always overruns |
| 4 | 6 | | |
