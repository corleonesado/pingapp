# pingapp

> Insider One DevOps internship case study, 4-day edition — **Track B** (local minikube + tunnel).

A tiny HTTP service in Go (~10MB distroless image) shipped end-to-end through Docker, Helm, minikube, GitHub Actions, and observability — exposed via a public tunnel. The point is **not** a perfect system; it is a small, reproducible slice with clear decisions documented as ADRs.

The brief lives in [`docs/case-study.pdf`](docs/case-study.pdf). The plan with live checkboxes lives in [`ROADMAP.md`](ROADMAP.md).

---

## Status

| Day | Theme | State |
|---|---|---|
| 1 | Foundation — app, container, repo | in progress |
| 2 | Kubernetes & Helm | pending |
| 3 | CI/CD & supply-chain security | pending |
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

Planned: ADR-0002 (Helm), ADR-0003 (auto-deploy strategy), ADR-0004 (Trivy thresholds), ADR-0005 (tunnel choice).

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
