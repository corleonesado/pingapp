# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- ArgoCD installed on the local minikube; `Application` synced against `ghcr.io/corleonesado/pingapp:v0.1.0` with `automated` + `prune` + `selfHeal`.
- Evidence: `docs/screenshots/day3-01-argocd-initial-sync.txt`, `day3-02-argocd-gitops-rollout.txt` (chart change → sync in ~20s).

### Changed
- `Makefile` `argocd-install` uses `--server-side --force-conflicts` to handle ArgoCD's large CRD annotations.
- `values-prod.yaml`: `replicaCount` 2 → 3 (demo of the GitOps loop).

## [0.1.0] - 2026-05-21

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
- CI workflow (`.github/workflows/ci.yml`): test (race + cover), golangci-lint, gitleaks, docker build, Trivy scan (fail on HIGH/CRITICAL), and a main-only GHCR push of the immutable `:<sha>` tag.
- Release workflow (`.github/workflows/release.yml`): on `v*.*.*` tag — build, Trivy scan, push `:<semver>` + `:<sha>`, generate an SPDX SBOM with Syft, and create a GitHub Release with notes from this changelog.
- `.golangci.yml` (golangci-lint v2) and a package comment to satisfy `revive`.
- ArgoCD GitOps wiring under `deploy/argocd/` (Application + bootstrap docs) and Makefile targets `argocd-install`, `argocd-password`, `argocd-ui`, `argocd-app`.
- ADR-0001: Language and runtime (Go on distroless).
- ADR-0002: Helm over raw manifests / Kustomize.
- ADR-0003: Auto-deploy on merge via ArgoCD GitOps (pull model).
- ADR-0004: Image scanning gate (Trivy on HIGH/CRITICAL, ignore-unfixed).

### Changed
- Build toolchain bumped to `golang:1.26-alpine` so the compiled binary picks up Go stdlib security fixes; clears all Trivy HIGH/CRITICAL findings. The go.mod language floor stays at 1.22.
