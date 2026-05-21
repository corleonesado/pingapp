# ADR-0002: Helm over raw manifests and Kustomize

- **Status**: Accepted
- **Date**: 2026-05-20
- **Deciders**: Sadettin Er

## Context

The case study brief explicitly requires a Helm chart, so the "no packaging tool at all" option is off the table. But the choice still matters: Helm vs Kustomize, single chart vs umbrella chart, `helm create` skeleton vs hand-rolled. Day 2 needs a `dev` and a `prod` environment that visibly differ (replica count, host, resources), plus probes pointed at `/healthz`, plus a rollout/rollback demo. The packaging tool has to support all of that idiomatically and feed cleanly into Day 3 CI (image tag = git SHA on PR, semver on release).

## Decision

A single Helm chart at `charts/pingapp` (started from `helm create` then trimmed), with environment-specific values in `values-dev.yaml` and `values-prod.yaml`. Templates: `deployment.yaml`, `service.yaml`, `ingress.yaml`, `configmap.yaml`, `pdb.yaml`, `_helpers.tpl`. Image tag is parameterized so CI can drive it.

## Alternatives considered

- **Raw manifests + `kubectl apply -k`** (Kustomize) — fine for static structure, but the brief explicitly asks for Helm. Kustomize also makes the rollout/rollback story messier: there's no `helm history` / `helm rollback` equivalent, you'd diff revisions manually.
- **Helm + Kustomize together** (`helm template | kubectl apply -k`) — common in larger orgs but overkill here and explicitly flagged as "don't suggest" in CLAUDE.md. Two tools, one job.
- **Umbrella chart with subcharts for app + monitoring** — premature: Day 4's Prometheus stack is installed from its own upstream chart, not as a subchart. A flat chart keeps `helm lint` clean.
- **Hand-roll templates from scratch** (skip `helm create`) — slightly cleaner output but loses the conventional `_helpers.tpl` labels (`app.kubernetes.io/name`, `instance`, `version`, `managed-by`) that downstream tools like ArgoCD and kube-prometheus-stack's ServiceMonitor selector expect.

## Consequences

- **Positive:** `helm upgrade --install -f values-{dev,prod}.yaml` is one command per environment — the case study's Day 2 deliverable lands naturally.
- **Positive:** `helm history` and `helm rollback` give a first-class rollout audit trail, which is the screenshot evidence the brief asks for.
- **Positive:** Conventional `app.kubernetes.io/*` labels mean the Day 4 `ServiceMonitor` selector works without label gymnastics.
- **Positive:** Day 3 CI can override `image.tag` with `--set image.tag=$SHA` without touching files in git.
- **Trade-off accepted:** Helm's templating uses Go text/template, which is whitespace-sensitive and easy to write subtle bugs in. Mitigated by `helm lint` and `helm template -f values-dev.yaml | kubeconform` in CI (Day 3).
- **Trade-off accepted:** Three replicas across dev/prod means at least one PodDisruptionBudget; chose `minAvailable: 1` to keep dev viable.
- **Risk to revisit:** If we add a second service later, this single chart becomes a constraint; the migration to a library chart or an umbrella is straightforward but not free.

## Notes

- Trimmed from the `helm create` skeleton: `templates/tests/`, `serviceaccount.yaml` (chart runs as the cluster default, locked down by `securityContext`), and the autoscaling/HPA stub (we may add it as the Day 2 bonus separately).
- `appVersion` in `Chart.yaml` is updated alongside the image tag at release time; not coupled to chart `version` so chart-only changes (e.g., probe tuning) can bump the chart without forcing a re-release of the app.
