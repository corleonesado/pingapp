# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Tiny HTTP service in Go with `/ping`, `/healthz`, and `/version` endpoints.
- Structured JSON logging via `log/slog`.
- Request ID middleware (UUID v4 per request, propagated as `X-Request-ID`).
- Multi-stage `Dockerfile` producing a `gcr.io/distroless/static:nonroot` image (~14MB, non-root UID 65532).
- `docker-compose.yaml` for local development (read-only fs, dropped caps).
- `Makefile` with `run`, `test`, `lint`, `docker-build`, `docker-run` targets.
- Helm chart at `charts/pingapp/` with Deployment, Service, Ingress, ConfigMap, PodDisruptionBudget templates.
- `values-dev.yaml` (1 replica, debug logs, `dev.pingapp.local`) and `values-prod.yaml` (2 replicas, info logs, `pingapp.local`, PDB enabled).
- Liveness + readiness probes on `/healthz`; optional startup probe behind a flag.
- Container security context: `runAsNonRoot`, `readOnlyRootFilesystem`, `cap drop ALL`, `seccompProfile: RuntimeDefault`.
- Makefile targets `minikube-up/down`, `minikube-load`, `helm-lint`, `helm-template-{dev,prod}`, `deploy-dev`, `deploy-prod`, `rollback`, `history`, `status`, `uninstall`.
- ADR-0001: Language and runtime (Go on distroless).
- ADR-0002: Helm over raw manifests / Kustomize.
