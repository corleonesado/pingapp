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
- [ ] Branch protection on `main`: require PR, require CI green (set up after CI exists)
- [ ] First 2-3 conventional commits (commands provided; user to run)

### 1.4 Decisions to capture
- [x] ADR-0001: language and runtime (Go + distroless, covers base image rationale)

### Day 1 checkpoint
- [x] `docker build` and `docker run` succeed cleanly
- [x] `curl localhost:8080/ping` returns `pong`
- [x] `curl localhost:8080/healthz` returns 200
- [x] `curl localhost:8080/version` returns the SHA (currently `dev` — will be a real SHA once first commit is made)
- [ ] `git log` is clean (conventional commits, no secrets) — pending first commit
- [ ] `gitleaks detect --no-git` passes locally — pending gitleaks install

---

## Day 2 — Kubernetes & Helm

> **Deliverable**: `helm upgrade --install` works against minikube with visible dev vs prod differences, rollback proven.

### 2.1 Helm chart
- [ ] `helm create charts/pingapp` then trim the noise (`tests/`, `serviceaccount.yaml` if not needed)
- [ ] Templates: `deployment.yaml`, `service.yaml`, `ingress.yaml`, `configmap.yaml`, `_helpers.tpl`
- [ ] `Chart.yaml` with `appVersion` aligned to image tag
- [ ] `image.repository` and `image.tag` parameterized
- [ ] `helm lint charts/pingapp` passes

### 2.2 Environments
- [ ] `values-dev.yaml`: 1 replica, low resources, host `dev.pingapp.local`
- [ ] `values-prod.yaml`: 2-3 replicas, higher resources, host `pingapp.local`
- [ ] `/etc/hosts` entries documented in README for local ingress
- [ ] Diff documented in README (table form)

### 2.3 Probes & resources
- [ ] `livenessProbe.httpGet.path: /healthz`, `initialDelaySeconds: 5`, `periodSeconds: 10`
- [ ] `readinessProbe.httpGet.path: /healthz`, `periodSeconds: 5`
- [ ] (Optional) `startupProbe` for slow-start safety
- [ ] `resources.requests`: CPU 50m, memory 32Mi (tune by observation)
- [ ] `resources.limits`: CPU 200m, memory 64Mi
- [ ] Document the reasoning briefly in README

### 2.4 Rollout & rollback
- [ ] Build a `v0.1.1` image, push into minikube (`minikube image load`)
- [ ] `helm upgrade pingapp ... --set image.tag=v0.1.1`
- [ ] Capture `kubectl rollout status deployment/pingapp` output
- [ ] `helm rollback pingapp 1`
- [ ] Capture `helm history pingapp`
- [ ] Save screenshots to `docs/screenshots/day2-*.png`

### 2.5 Bonus (pick **one** if time allows)
- [ ] HPA (CPU-based, simple target)
- [ ] NetworkPolicy (allow only ingress-controller → app)
- [ ] PodDisruptionBudget

### 2.6 Decisions to capture
- [ ] ADR-0002: Helm over raw manifests / Kustomize

### Day 2 checkpoint
- [ ] `make deploy-dev` → pods Running, probes Healthy
- [ ] `make deploy-prod` → different replica count and host visible
- [ ] Rollback demonstrated and screenshotted
- [ ] Ingress reachable via `curl --resolve dev.pingapp.local:80:<minikube-ip> http://dev.pingapp.local/ping`

---

## Day 3 — CI/CD & supply-chain security

> **Deliverable**: Green CI run, image on GHCR, `v0.1.0` GitHub Release, auto-deploy to minikube on merge to main.

### 3.1 CI pipeline (`.github/workflows/ci.yml`)
- [ ] Trigger: PRs to `main`, pushes to `main`
- [ ] Job: `test` — `go test ./... -race -cover`
- [ ] Job: `lint` — `golangci-lint run`
- [ ] Job: `build` — docker buildx with cache, tag `:<sha>`
- [ ] Job: `scan` — Trivy on the built image, fail on HIGH/CRITICAL
- [ ] Job: `push` (main only) — push to `ghcr.io/corleonesado/pingapp:<sha>`
- [ ] Use `actions/cache` for Go modules and Docker layers
- [ ] Concurrency group to cancel stale runs

### 3.2 Secrets & auth
- [ ] GHCR push uses `${{ secrets.GITHUB_TOKEN }}` with `packages: write` permission
- [ ] No PATs anywhere
- [ ] `gitleaks/gitleaks-action` step in CI
- [ ] Verify: `grep -r "ghp_\|AKIA\|password" .` returns nothing real

### 3.3 Release hygiene
- [ ] `CHANGELOG.md` in Keep a Changelog format with `[Unreleased]`
- [ ] `.github/workflows/release.yml`: trigger on `v*.*.*` tag push
- [ ] Release workflow: build, scan, push image with `:v0.1.0` and `:<sha>`
- [ ] Create GitHub Release with notes from CHANGELOG
- [ ] Cut `v0.1.0`

### 3.4 Auto-deploy on merge
- [ ] **Decision**: ArgoCD GitOps (recommended) or pipeline `kubectl set image`
- [ ] If ArgoCD: install on minikube, create `Application` pointing at the Helm chart, sync policy auto
- [ ] If pipeline: self-hosted runner OR `kubectl` via context file (document the risk)
- [ ] Verify: merge a PR with an image bump → new pod rolls out within ~30s

### 3.5 Bonus (pick **one** if time allows)
- [ ] cosign keyless signing of the image (OIDC)
- [ ] Syft SBOM attached to the release
- [ ] Multi-arch build (amd64 + arm64) via buildx

### 3.6 Decisions to capture
- [ ] ADR-0003: auto-deploy strategy (ArgoCD vs pipeline-kubectl)
- [ ] ADR-0004: image scanning gate (Trivy thresholds)

### Day 3 checkpoint
- [ ] Open a test PR → CI green end-to-end
- [ ] Image visible at `ghcr.io/corleonesado/pingapp`
- [ ] `v0.1.0` GitHub Release exists with notes
- [ ] Merge to main triggers a new pod rollout in minikube
- [ ] gitleaks step passes, Trivy step passes

---

## Day 4 — Observability & docs

> **Deliverable**: Grafana dashboard + alert, public URL, RUNBOOK, SECURITY, ADRs, architecture diagram.

### 4.1 Logs & metrics (app side)
- [ ] Confirm structured JSON logs from Day 1 still emit
- [ ] Add `/metrics` endpoint using `prometheus/client_golang`
- [ ] Counters: `http_requests_total{method,path,status}`
- [ ] Histogram: `http_request_duration_seconds{method,path}`
- [ ] Gauge: `http_requests_in_flight`
- [ ] Unit test for the metrics middleware

### 4.2 Prometheus + Grafana
- [ ] `helm repo add prometheus-community ...`
- [ ] `helm install kps prometheus-community/kube-prometheus-stack -n monitoring --create-namespace`
- [ ] `ServiceMonitor` for the app (add to chart)
- [ ] Grafana dashboard panels: RPS, latency p50/p95, error rate, pod restarts
- [ ] `PrometheusRule`: alert if `error_rate > 5%` for 5m
- [ ] Verify alert fires (cause errors transiently) and resolves
- [ ] Save dashboard JSON in `docs/grafana-dashboard.json`
- [ ] Screenshots → `docs/screenshots/day4-*.png`

### 4.3 Reproducibility (Track B IaC substitute)
- [ ] `Makefile` covers every workflow command from CLAUDE.md
- [ ] `scripts/bootstrap.sh`:
  - Checks for `docker`, `kubectl`, `helm`, `minikube`, `cloudflared`
  - Prints install hints for missing ones
  - Idempotent (safe to re-run)
- [ ] README documents: "Fresh laptop → working demo in N minutes"

### 4.4 Architecture & docs
- [ ] `docs/architecture.png` (Excalidraw): laptop → docker → minikube → ingress → cloudflared → public URL, with Prometheus/Grafana sidecar overlay
- [ ] `RUNBOOK.md`: one-page incident guide
  - How to restart the app
  - Where to find logs (`kubectl logs`, Grafana log panel if Loki added)
  - How to roll back (`helm rollback`)
  - How to rotate a secret
  - Common failures + fixes
- [ ] `SECURITY.md`:
  - Threat model (one paragraph)
  - Secret handling policy
  - Vulnerability reporting contact
  - Image scanning policy
- [ ] **At least 3 ADRs** in `docs/decisions/` (target: 5 — add ADR-0005 for tunnel choice if time)

### 4.5 Public URL
- [ ] `cloudflared tunnel --url http://<ingress-host>:<port>` (or `kubectl port-forward` target)
- [ ] Document the URL in README (note: ephemeral for quick tunnels)
- [ ] Demo: record a short gif/video of `curl <public-url>/ping` → `pong`
- [ ] If time: register a custom subdomain and run a named tunnel

### Day 4 checkpoint
- [ ] Grafana dashboard renders, at least one alert defined
- [ ] RUNBOOK reads as a one-page incident guide
- [ ] ADRs cover "why Helm", "why this base image", "why this tunnel"
- [ ] Public URL reachable from outside your network
- [ ] Repo is "fresh-laptop reproducible" via `make bootstrap`

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

- [ ] Public GitHub repo link (or `insider-one-devops` invited if private)
- [ ] `README.md` with setup, run, env vars, architecture notes, track chosen
- [ ] Architecture diagram in repo (Excalidraw export OK)
- [ ] Helm chart folder with `values-dev.yaml` and `values-prod.yaml`
- [ ] `.github/workflows/` with green run evidence (link to a successful run)
- [ ] Screenshots: `kubectl get pods`, `helm list`, `helm history`, `kubectl rollout status`
- [ ] Grafana screenshot with at least one dashboard + alert visible
- [ ] Public URL or demo video of `/ping` returning `pong`
- [ ] `RUNBOOK.md` and `SECURITY.md`
- [ ] **3+ ADRs**, each 3-5 sentences (target 4-5 ADRs)

---

## Time budget

| Day | Hours (target) | Hours (actual) | Notes |
|---|---|---|---|
| 1 | 6 | | |
| 2 | 6 | | |
| 3 | 7 | | CI debugging always overruns |
| 4 | 6 | | |
