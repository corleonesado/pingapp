# ADR-0003: Auto-deploy on merge — ArgoCD GitOps (pull model)

- **Status**: Accepted
- **Date**: 2026-05-21
- **Deciders**: Sadettin Er

## Context

We chose Track B: minikube runs locally on a laptop, behind NAT, with no inbound reachability. Day 3 asks that a merge to `main` results in a new pod rolling out on the cluster. The hard constraint is network direction: a GitHub-hosted Actions runner **cannot reach** a laptop-local Kubernetes API. Any "push from CI" model (a `kubectl set image` step in the pipeline) would require either exposing the cluster to the internet or running a self-hosted runner on the laptop.

## Decision

GitOps with ArgoCD running **inside** the local minikube. ArgoCD reaches *out* to GitHub, watches `charts/pingapp` on `main`, and auto-syncs the cluster to match the chart. Deployment becomes a pull, not a push — which is exactly what works behind NAT.

## Alternatives considered

- **Pipeline `kubectl set image` from GitHub Actions** — needs inbound access to the cluster API. For Track B that means tunneling the API server to the internet (large attack surface, flagged as a risk in the brief) or a self-hosted runner.
- **Self-hosted GitHub Actions runner on the laptop** — works and keeps the push model, but runs arbitrary workflow code on a personal machine and must be online for deploys. Acceptable for some teams; too heavy and security-awkward for this scope.
- **Flux instead of ArgoCD** — equally valid GitOps pull model. Chose ArgoCD for its UI, which makes the sync state and rollout visibly demonstrable for the case study screenshots. (Noted as the alternative in CLAUDE.md.)
- **No auto-deploy, manual `make deploy-prod`** — honest and low-effort, but a weaker signal than a working GitOps loop.

## Consequences

- **Positive:** Works behind NAT with zero inbound exposure — the cluster API never touches the internet.
- **Positive:** Git is the single source of truth; the live state is reconciled to the chart, and drift is auto-corrected (self-heal).
- **Positive:** ArgoCD's UI shows sync status, diff, and rollout history — good demo material.
- **Trade-off accepted:** ArgoCD needs read access to the (private) repo — provided via a least-privilege, read-only credential, not a broad PAT. It also needs to pull the image, so the GHCR package is made public (read-only) to avoid a registry pull secret; nothing sensitive is in the image.
- **Trade-off accepted:** Image-tag promotion is not automatic out of the box. For this scope, a merge that changes the chart/values (including the pinned image tag) triggers a sync. Fully automated tag bumps would need ArgoCD Image Updater or a CI write-back commit — noted as a future enhancement, not built now.
- **Risk to revisit:** ArgoCD only deploys while minikube + ArgoCD are running on the laptop. This is a local-demo constraint, not a production posture; a real cluster would run ArgoCD centrally.

## Notes

- Sync policy: `automated` with `prune: true` and `selfHeal: true`.
- The Application targets `charts/pingapp` with `values-prod.yaml`, namespace `default`.
- Bootstrap commands and the `Application` manifest live under `deploy/argocd/` and are wired into the Makefile (`argocd-install`, `argocd-app`).
