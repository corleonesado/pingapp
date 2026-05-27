# CLAUDE.md

Context file for Claude Code working on the pingapp repo.

> **Read this first, every session.** Then check the current state of `main` (`git log`, `make status`, open PRs) before changing anything.

---

## What this is

A tiny HTTP service in Go shipped end-to-end through Docker, Helm, minikube, GitHub Actions, ArgoCD GitOps, and a Prometheus/Grafana observability stack — exposed via a public cloudflared tunnel. **Track B** of the Insider One DevOps internship case study (local minikube + tunnel, no AWS).

The brief: `docs/case-study.pdf`. The shipped behaviour and operating instructions live in `README.md`, `RUNBOOK.md`, and `SECURITY.md`. Decisions live in `docs/decisions/`.

The goal is **not** a perfect system. It is a small, reproducible slice with clear decisions. Every non-obvious choice gets an ADR.

---

## Tech stack (chosen — rationale in `docs/decisions/`)

| Layer | Choice | Why (one-liner) |
|---|---|---|
| Language | Go 1.22+ in source, built with 1.26 toolchain | Single static binary, tiny image, patched stdlib |
| HTTP | stdlib `net/http` | No framework lock-in, this service is tiny |
| Container | Multi-stage → `gcr.io/distroless/static:nonroot` | No shell, no package manager, smallest attack surface |
| Local | `docker-compose.yaml` | One command for new contributors |
| Orchestration | minikube (Docker driver) | Track B requirement |
| Packaging | Helm chart in `charts/pingapp` | Required by case study |
| CI/CD | GitHub Actions → GHCR | Free, OIDC-capable, native to repo |
| Image scan | Trivy, fail on HIGH/CRITICAL | Required by case study |
| Secret scan | gitleaks | Catches accidental commits |
| Workflow lint | actionlint | Catches workflow bugs locally |
| Dep hygiene | Dependabot (Go, Actions, Docker) | Weekly automated bumps |
| Metrics | `prometheus/client_golang` + kube-prometheus-stack | Industry default |
| Tunnel | cloudflared quick tunnel | More stable than ngrok free tier |
| IaC (Track B) | Makefile + `scripts/bootstrap.sh` | Track B substitute for Terraform |
| Auto-deploy | ArgoCD (GitOps pull model) | Works behind NAT, stronger signal than `kubectl set image` |

---

## Service contract

Endpoints:
- `GET /ping` → `pong` (200)
- `GET /healthz` → `200 OK` (used by K8s probes)
- `GET /version` → JSON with build SHA, injected via `-ldflags`
- `GET /metrics` → Prometheus exposition (counters, histogram, gauge, Go runtime collectors)
- `GET /chaos` → 500, registered only when `ENABLE_CHAOS=1` (alert drills)

Logs are structured JSON: `timestamp`, `level`, `msg`, `request_id`, `method`, `path`, `status`, `duration`. A request-id middleware tags every request.

Config is env-driven only:
- `PORT` (default 8080)
- `LOG_LEVEL` (default info)
- `ENABLE_CHAOS` (default 0)
- `VERSION` (injected at build)

---

## Project structure

```
.
├── CLAUDE.md                   # this file
├── README.md                   # entry point
├── RUNBOOK.md, SECURITY.md     # operating + security docs
├── CHANGELOG.md                # Keep a Changelog format
├── Makefile                    # primary task runner
├── Dockerfile, docker-compose.yaml, .dockerignore
├── .env.example, .gitignore, .golangci.yml
├── cmd/server/main.go          # entrypoint
├── internal/
│   ├── handlers/               # HTTP handlers + middleware + tests
│   ├── logger/                 # structured logger (slog)
│   └── metrics/                # prom client_golang collectors + middleware
├── charts/pingapp/
│   ├── Chart.yaml, values.yaml, values-dev.yaml, values-prod.yaml
│   └── templates/              # Deployment, Service, Ingress, ConfigMap,
│                               #   PDB, ServiceMonitor, PrometheusRule, _helpers
├── deploy/argocd/              # Application + setup notes
├── scripts/bootstrap.sh        # fresh-laptop toolchain check
├── .github/
│   ├── workflows/ci.yml, release.yml
│   ├── dependabot.yml
│   ├── CODEOWNERS, PULL_REQUEST_TEMPLATE.md
└── docs/
    ├── case-study.pdf, architecture.png, grafana-dashboard.json
    ├── postmortem.md
    ├── decisions/              # ADRs 0001–0005
    └── screenshots/            # evidence logs
```

---

## Conventions

- **Commits**: Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`, `ci:`, `refactor:`).
- **Branches**: `main` is the integration branch. Non-trivial changes go through a PR on `feat/<short-desc>`.
- **Versioning**: Semver tags (`v0.1.0`). Image tag = git SHA in CI, semver on release.
- **No `:latest`** anywhere — manifests, compose, Helm values, nowhere.
- **PR template**: required. Self-review checklist before requesting review.

---

## Key commands

```bash
# local
make run                 # go run with hot defaults
make test                # go test ./... -race -cover
make lint                # golangci-lint
make docker-build        # multi-stage build with VERSION arg
make docker-run          # run + port forward
make bootstrap           # verify local toolchain

# kubernetes
make minikube-up         # cluster + ingress + metrics-server
make deploy-dev          # helm upgrade --install -f values-dev.yaml
make deploy-prod
make rollback            # helm rollback to previous revision
make argocd-install      # ArgoCD into argocd namespace
make argocd-app TAG=...  # apply the Application

# observability
make obs-install         # helm install kube-prometheus-stack
make grafana             # port-forward grafana (admin/admin)

# public URL
make tunnel              # cloudflared quick tunnel — prints URL
```

---

## Security non-negotiables

- **No secrets in repo**, ever. gitleaks runs in CI; if it fails, fix the leak and **rotate** the credential.
- **No `:latest` tags** in manifests, compose, or values files.
- **Non-root container**: distroless `nonroot` sets UID 65532. Don't override.
- **Trivy gate**: pipeline fails on HIGH/CRITICAL. Don't disable — bump base image or pin a fixed version.
- **`.env.example` only** in git. Real `.env` is gitignored.
- **Probes** point at `/healthz`, never `/ping`.
- **Ingress** is the only externally reachable surface. NodePorts are dev-only.
- **`/chaos`** stays off in prod. Enable only as a time-boxed drill via `--set chaos.enabled=true`.

---

## Workflow Claude Code should follow

1. **Find the current task** from the user or from open PRs / issues / `git log`.
2. **One concern per change.** Small, reviewable diffs.
3. **When a decision is non-obvious** (tool choice, structural shift, security trade-off), write or update an ADR in `docs/decisions/` using `0000-adr-template.md`.
4. **User-facing changes** belong in `CHANGELOG.md` under `## [Unreleased]`.
5. **Before declaring done**: `make test && make lint`.
6. **Helm changes**: render locally with `helm template charts/pingapp -f charts/pingapp/values-dev.yaml` and skim the output.
7. **Never invent secrets, URLs, or account IDs.** Use `<placeholder>` and let the user fill in.

---

## What Claude Code should NOT do

- Don't write raw Kubernetes manifests once the Helm chart exists — template it.
- Don't suggest "we could use Kustomize and Helm together" — pick one.
- Don't pull in heavy frameworks (Gin, Echo, gRPC). The service is meant to be tiny.
- Don't paste real credentials anywhere. Use `<placeholder>` or env var references.
- Don't disable failing security checks. Fix the root cause.

---

## AI assistance disclosure

Per the case study's house rule: Claude (chat) and Claude Code are used throughout this project. A short summary of AI usage lives in `README.md` under "AI usage", and significant decisions made with AI input are captured in their respective ADRs.
