# pingapp

> Insider One DevOps internship case study, 4-day edition — **Track B** (local minikube + tunnel).

A tiny HTTP service in Go (~10MB distroless image) shipped end-to-end through Docker, Helm, minikube, GitHub Actions, and observability — exposed via a public tunnel. The point is **not** a perfect system; it is a small, reproducible slice with clear decisions documented as ADRs.

The brief lives in [`docs/case-study.pdf`](docs/case-study.pdf). The plan with live checkboxes lives in [`ROADMAP.md`](ROADMAP.md).

---

## Status

| Day | Theme | State |
|---|---|---|
| 1 | Foundation — app, container, repo | done |
| 2 | Kubernetes & Helm | done |
| 3 | CI/CD & supply-chain security | done |
| 4 | Observability & docs | pending |

---

## Endpoints

| Method | Path | Response | Purpose |
|---|---|---|---|
| GET | `/ping` | `pong` (text/plain, 200) | Human/demo endpoint |
| GET | `/healthz` | `OK` (text/plain, 200) | Kubernetes liveness/readiness |
| GET | `/version` | `{"version":"<sha>"}` (application/json) | Build identity, injected at build |
| GET | `/metrics` | Prometheus exposition | **Day 4** — not yet wired |

Every response includes an `X-Request-ID` header. Incoming `X-Request-ID` headers are honored; otherwise a UUID v4 is generated. The same id is attached to the structured JSON access log line for that request.

---

## Quick start

### Run locally with Go

```bash
make run                 # listens on :8080, version = current git SHA
curl localhost:8080/ping # -> pong
```

### Run via Docker

```bash
make docker-build        # builds pingapp:<sha> and pingapp:dev
make docker-run          # forwards 8080 -> 8080
curl localhost:8080/ping
```

### Run via docker compose

```bash
docker compose up --build -d
curl localhost:8080/ping
docker compose down
```

### Tests

```bash
make test                # go test ./... -race -cover
```

---

## Configuration

All config is env-driven; see [`.env.example`](.env.example).

| Var | Default | Notes |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `VERSION` | (build-time) | Injected via `-ldflags "-X main.version=…"`, not read at runtime |

---

## Kubernetes (Day 2)

The Helm chart lives in [`charts/pingapp/`](charts/pingapp/). One chart, two values files; the chart is hand-rolled (not from `helm create`) — rationale in [ADR-0002](docs/decisions/0002-helm-over-raw-manifests.md).

### Quick start

```bash
make minikube-up     # minikube start + ingress + metrics-server addons
make deploy-dev      # builds image, loads into minikube, helm upgrade --install with dev values
make status          # pods, svc, ingress, rollout status

# /etc/hosts (one-time, optional — or use --resolve below):
echo "$(minikube ip) dev.pingapp.local pingapp.local" | sudo tee -a /etc/hosts

curl --resolve dev.pingapp.local:80:$(minikube ip) http://dev.pingapp.local/ping
```

### Dev vs prod

|  | dev (`values-dev.yaml`) | prod (`values-prod.yaml`) |
|---|---|---|
| Replicas | **1** | **2** |
| Ingress host | `dev.pingapp.local` | `pingapp.local` |
| Log level | `debug` | `info` |
| CPU request / limit | 25m / 100m | 50m / 200m |
| Memory request / limit | 16Mi / 32Mi | 32Mi / 64Mi |
| PodDisruptionBudget | disabled (1 replica) | `minAvailable: 1` |

Probes (same in both): liveness on `/healthz` every 10s after 5s grace; readiness on `/healthz` every 5s after 2s. `restartPolicy` is the deployment default (`Always`). The container drops all Linux capabilities, runs as UID 65532, and uses a read-only root filesystem with `seccompProfile: RuntimeDefault`.

### Rollout / rollback

```bash
make deploy-dev                           # rev 1
make deploy-prod                          # rev 2 — replicas 1→2, dev→prod host
make history                              # see helm history
make rollback                             # back to rev 2 (or any earlier with `helm rollback pingapp <n>`)
```

Evidence from a clean run is captured in [`docs/screenshots/`](docs/screenshots/) (text logs of `kubectl`, `helm history`, and ingress curls — screenshots come on Day 4 with Grafana panels).

---

## CI/CD & supply chain (Day 3)

Two workflows under [`.github/workflows/`](.github/workflows/):

**`ci.yml`** — on PRs and pushes to `main`:

| Job | What |
|---|---|
| `test` | `go test ./... -race -cover` (module + build cache) |
| `lint` | `golangci-lint` (config in `.golangci.yml`) |
| `gitleaks` | secret scan across full history |
| `build & scan` | docker build → **Trivy** (fails on HIGH/CRITICAL, `--ignore-unfixed`) → push `:<sha>` to GHCR **(main only)** |

Concurrency cancels stale runs per ref. GHCR push uses the built-in `GITHUB_TOKEN` with `packages: write` — **no PATs**. Only the immutable `:<sha>` tag is pushed; there are no `:latest` tags anywhere ([ADR-0004](docs/decisions/0004-image-scanning-gate.md)).

**`release.yml`** — on a `v*.*.*` tag: build → Trivy scan → push `:<semver>` + `:<sha>` → generate an SPDX **SBOM** with Syft → create a GitHub Release with notes pulled from `CHANGELOG.md` and the SBOM attached.

### Auto-deploy — ArgoCD GitOps

A GitHub-hosted runner can't reach a laptop-local minikube, so deploy is a **pull**: ArgoCD runs inside the cluster and syncs `charts/pingapp` from this repo. Setup and rationale: [`deploy/argocd/`](deploy/argocd/) and [ADR-0003](docs/decisions/0003-auto-deploy-argocd.md).

```bash
make argocd-install          # ArgoCD into the argocd namespace
make argocd-app TAG=v0.1.0   # Application pointed at the GHCR image
```

---

## Architecture (Day 1 slice)

```
        ┌─────────────┐
curl ───▶ docker run  ├──▶ :8080 ─▶ /ping, /healthz, /version
        │  pingapp    │
        └─────────────┘
              │
              ▼
        stdout JSON logs (slog)
```

Days 2–4 will extend this into: docker → minikube (Helm chart, ingress) → cloudflared tunnel → public URL, with Prometheus + Grafana scraping `/metrics`. The full diagram lives in `docs/architecture.png` (Day 4 deliverable).

---

## Decisions

ADRs live in [`docs/decisions/`](docs/decisions/). Currently:

- [ADR-0001 — Language and runtime: Go on distroless](docs/decisions/0001-language-and-runtime.md)
- [ADR-0002 — Helm over raw manifests / Kustomize](docs/decisions/0002-helm-over-raw-manifests.md)
- [ADR-0003 — Auto-deploy via ArgoCD GitOps](docs/decisions/0003-auto-deploy-argocd.md)
- [ADR-0004 — Image scanning gate (Trivy HIGH/CRITICAL)](docs/decisions/0004-image-scanning-gate.md)

Planned: ADR-0005 (tunnel choice, Day 4).

---

## Project layout

See [`CLAUDE.md`](CLAUDE.md) for the full tree and conventions. Highlights:

- `cmd/server/` — entrypoint (`main.go`)
- `internal/handlers/` — HTTP handlers + middleware + tests
- `internal/logger/` — slog setup
- `charts/pingapp/` — Helm chart (Day 2)
- `.github/workflows/` — CI/CD (Day 3)
- `docs/decisions/` — ADRs

---

## AI Usage

Per the case study's house rule: this project uses Claude (chat) and Claude Code throughout. Claude Code is used for scaffolding source, writing tests, drafting ADRs, and editing docs; final review and decisions are mine. Significant decisions with AI input are captured in their respective ADRs.

---

## Submission

This is a private learning submission for the Insider One DevOps internship case study, May 2026.
