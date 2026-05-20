# CLAUDE.md

Context file for Claude Code working on the Insider One DevOps internship case study.

> **Read this first, every session.** Then check `ROADMAP.md` for the current state before doing anything.

---

## What we're building

A small HTTP service shipped end-to-end through Docker, Helm, minikube, GitHub Actions, and observability — exposed via a public tunnel. **Track B** (local minikube + tunnel, no AWS).

The brief: `docs/case-study.pdf`. The plan with live checkboxes: `ROADMAP.md`.

The goal is **not** a perfect system. It is a small, reproducible slice with clear decisions. Every non-obvious choice gets an ADR.

---

## Tech stack (chosen — rationale lives in `docs/decisions/`)

| Layer | Choice | Why (one-liner) |
|---|---|---|
| Language | Go 1.22+ | Single static binary, tiny image, fast iteration |
| HTTP | stdlib `net/http` | No framework lock-in, this service is tiny |
| Container | Multi-stage → `gcr.io/distroless/static:nonroot` | No shell, no package manager, smallest attack surface |
| Local | `docker-compose.yaml` | One command for new contributors |
| Orchestration | minikube (Docker driver) | Track B requirement |
| Packaging | Helm chart in `charts/pingapp` | Required by case study |
| CI/CD | GitHub Actions → GHCR | Free, OIDC-capable, native to repo |
| Image scan | Trivy, fail on HIGH/CRITICAL | Required by case study |
| Secret scan | gitleaks | Catches accidental commits |
| Metrics | `prometheus/client_golang` + kube-prometheus-stack | Industry default |
| Tunnel | cloudflared quick tunnel | More stable than ngrok free tier |
| IaC (Track B) | Makefile + `scripts/bootstrap.sh` | Track B substitute for Terraform |
| Auto-deploy | ArgoCD (GitOps) | Stronger signal than `kubectl set image` |

---

## Service contract

Endpoints:
- `GET /ping` → `pong` (200)
- `GET /healthz` → `200 OK` (used by K8s probes)
- `GET /version` → JSON with build SHA, injected via `-ldflags`
- `GET /metrics` → Prometheus format (Day 4)

Logs are structured JSON: `timestamp`, `level`, `msg`, `request_id`. A `request_id` middleware tags every request.

Config is env-driven only:
- `PORT` (default 8080)
- `LOG_LEVEL` (default info)
- `VERSION` (injected at build)

---

## Project structure

```
.
├── CLAUDE.md                   # this file
├── README.md
├── ROADMAP.md                  # progress tracker — keep updated
├── RUNBOOK.md                  # Day 4
├── SECURITY.md                 # Day 4
├── CHANGELOG.md                # Keep a Changelog format
├── Makefile                    # primary task runner
├── Dockerfile
├── docker-compose.yaml
├── .dockerignore
├── .env.example
├── .gitignore
├── cmd/server/main.go          # entrypoint
├── internal/
│   ├── handlers/               # http handlers + tests
│   ├── logger/                 # structured logger
│   └── metrics/                # prometheus collectors (Day 4)
├── charts/pingapp/
│   ├── Chart.yaml
│   ├── values.yaml
│   ├── values-dev.yaml
│   ├── values-prod.yaml
│   └── templates/
├── .github/
│   ├── workflows/ci.yml
│   ├── workflows/release.yml
│   ├── CODEOWNERS
│   └── PULL_REQUEST_TEMPLATE.md
├── docs/
│   ├── architecture.png        # excalidraw export
│   ├── case-study.pdf
│   ├── decisions/              # ADRs (numbered)
│   └── screenshots/            # kubectl/helm/grafana evidence
└── scripts/
    └── bootstrap.sh
```

---

## Conventions

- **Commits**: Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`, `ci:`, `refactor:`)
- **Branches**: `main` is protected. Work in `feat/<short-desc>` and PR in.
- **Versioning**: Semver tags (`v0.1.0`). Image tag = git SHA in CI, semver on release.
- **No `:latest`** anywhere — manifests, compose, Helm values, nowhere.
- **PR template**: required. Self-review checklist before requesting review.

---

## Key commands (target state — fill in as we go)

```bash
# local dev
make run                 # go run with hot defaults
make test                # go test ./... -race -cover
make lint                # golangci-lint run
make docker-build        # multi-stage build with VERSION, SHA args
make docker-run          # run + healthcheck
docker compose up        # full local stack

# kubernetes (track B)
make minikube-up         # start minikube + addons (ingress, metrics-server)
make minikube-down
make deploy-dev          # helm upgrade --install -f values-dev.yaml
make deploy-prod
make rollback            # helm rollback pingapp
make tunnel              # cloudflared quick tunnel → prints public URL

# observability
make obs-install         # helm install kube-prometheus-stack
make grafana             # port-forward grafana (login printed)

# release
make release VERSION=v0.1.0   # tag + push, triggers release workflow
```

---

## Security non-negotiables

- **No secrets in repo**, ever. gitleaks runs in CI; if it fails, fix the leak and rotate the credential.
- **No `:latest` tags** in manifests, compose, or values files.
- **Non-root container**: distroless `nonroot` sets UID 65532. Don't override.
- **Trivy gate**: pipeline fails on HIGH/CRITICAL. Don't disable — bump base image or pin a fixed version.
- **`.env.example` only** in git. Real `.env` is gitignored.
- **Probes** point at `/healthz`, never `/ping` (the latter is for humans).
- **Ingress** is the only externally reachable surface. NodePorts are dev-only.

---

## Workflow Claude Code should follow

1. **Read `ROADMAP.md`** to find the current task.
2. **One concern per change.** Small, reviewable diffs.
3. **When a decision is non-obvious** (tool choice, structural shift, security trade-off), write or update an ADR in `docs/decisions/` using `0000-adr-template.md`.
4. **Always update**:
   - `ROADMAP.md` checkboxes when finishing a task
   - `CHANGELOG.md` under `## [Unreleased]` for user-facing changes
5. **Before declaring done**: `make test && make lint`.
6. **Helm changes**: render locally with `helm template charts/pingapp -f charts/pingapp/values-dev.yaml` and skim the output.
7. **Never invent secrets, URLs, or AWS account IDs.** Use `<placeholder>` and let the user fill in.

---

## What Claude Code should NOT do

- Don't add features beyond the current day's scope without a note in `ROADMAP.md`.
- Don't write raw Kubernetes manifests once the Helm chart exists — template it.
- Don't suggest "we could use Kustomize and Helm together" — pick one.
- Don't pull in heavy frameworks (Gin, Echo, gRPC). The service is meant to be tiny.
- Don't paste real credentials anywhere. Use `<placeholder>` or env var references.
- Don't disable failing security checks. Fix the root cause.

---

## AI assistance disclosure

Per the case study's house rule: Claude (chat) and Claude Code are used throughout this project. A short summary of AI usage lives in `README.md` under "AI Usage", and significant decisions made with AI input are captured in their respective ADRs.
