# RUNBOOK

One-page operator guide for pingapp. Read it cold during an incident; nothing here assumes you wrote the code.

The app lives in the `default` namespace of a local minikube cluster, deployed by an ArgoCD `Application` (`pingapp` in the `argocd` namespace). The image is `ghcr.io/corleonesado/pingapp:<semver>`.

## Quick reference

```bash
make status                                 # pods, svc, ingress, rollout
helm history pingapp                        # release history (if Helm-managed)
kubectl -n argocd get app pingapp           # ArgoCD sync + health
kubectl -n monitoring get pods              # Prometheus / Grafana / Alertmanager
make grafana                                # open Grafana (admin / admin)
```

Endpoints:
- `/ping` — humans
- `/healthz` — K8s probes
- `/version` — build SHA / semver
- `/metrics` — Prometheus scrape target
- `/chaos` — synthetic 500 for alert-fire testing (intentional)

## Restart the app

A clean restart of all pods, fastest first:

```bash
kubectl rollout restart deployment/pingapp -n default
kubectl rollout status deployment/pingapp -n default
```

If pods are stuck `Pending` or `ImagePullBackOff`, check:
- `kubectl describe pod <pod>` — usually says exactly why.
- GHCR package visibility — must be **public** (or a pull secret must be wired in). Confirm at `https://github.com/users/corleonesado/packages/container/pingapp`.

## Find the logs

```bash
# tail the live app pods (structured JSON)
kubectl logs -l app.kubernetes.io/instance=pingapp -n default --tail=200 -f

# logs from the previous pod (after a crash loop)
kubectl logs <pod> -n default --previous

# ArgoCD sync errors
kubectl -n argocd get application pingapp -o jsonpath='{.status.conditions}' | python3 -m json.tool
```

Every request log line includes `request_id`, `method`, `path`, `status`, `duration`. Correlate by `request_id` if a client reports a failed request.

## Roll back the app

ArgoCD owns desired state. Two ways to roll back:

1. **Roll back in Git** (preferred — keeps Git as the source of truth):
   ```bash
   git revert <bad-commit> && git push
   # ArgoCD auto-syncs within ~30s
   ```
2. **Roll back the image tag in `deploy/argocd/application.yaml`**:
   bump `image.tag` back to the last-known-good (`v0.1.0`, `v0.1.1`, ...), commit, push.

If ArgoCD itself is broken and you need an immediate fix, you can fall back to Helm (drift will be reconciled later by ArgoCD when it recovers):

```bash
make rollback                  # helm rollback to previous revision
helm history pingapp           # confirm the new revision is deployed
```

## Roll back ArgoCD's view (if a chart change broke the deploy)

```bash
kubectl -n argocd get app pingapp -o yaml | less        # inspect desired state
kubectl -n argocd patch app pingapp --type=merge \
  -p '{"operation":{"sync":{"revision":"<good-commit-sha>"}}}'
```

## Rotate a secret

The only "secret" in this project is the read-only PAT ArgoCD uses to read the repo, stored as `pingapp-repo` in the `argocd` namespace.

```bash
kubectl -n argocd delete secret pingapp-repo

stty -echo; printf "Paste new PAT (input hidden): " >&2; read PAT; stty echo; echo
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: pingapp-repo
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: repository
stringData:
  type: git
  url: https://github.com/corleonesado/pingapp.git
  username: corleonesado
  password: $PAT
EOF
unset PAT
```

Then **revoke the old PAT** at https://github.com/settings/personal-access-tokens.

## Common failures

| Symptom | Likely cause | Fix |
|---|---|---|
| `ImagePullBackOff` on app pods | GHCR package not public | Flip visibility in package settings, then `kubectl rollout restart deploy/pingapp`. |
| ArgoCD `OutOfSync` with `auth required` | PAT in `pingapp-repo` expired or revoked | Rotate (see above). |
| ArgoCD `OutOfSync` with `ServiceMonitor not registered` | kube-prometheus-stack is not installed | `helm upgrade --install kps prometheus-community/kube-prometheus-stack -n monitoring --create-namespace`. |
| `PingappHighErrorRate` alert firing in Grafana | Real spike *or* `/chaos` was used recently | Check `http_requests_total` by status; if dominated by `/chaos`, it's a drill. |
| Trivy gate fails CI on `main` with no code change | New CVE disclosed in the base image / Go stdlib | Bump the Dockerfile `ARG GO_VERSION` and re-run; do **not** disable the gate. |
| `helm upgrade` says "release exists" but ArgoCD is also managing | Stale Helm release left from manual deploy | `helm uninstall pingapp -n default --keep-history=false`; ArgoCD will recreate. |
| `make argocd-install` fails with `annotations Too long` | kubectl can't store ArgoCD's huge CRD last-applied annotation | Already fixed in the Makefile to use `--server-side --force-conflicts`. |

## Tunnel down

```bash
pgrep -f "cloudflared tunnel" || echo "no tunnel running"
make tunnel                    # restart; prints a NEW URL (quick tunnels are ephemeral)
```

Update any pinned demo URL in the submission email if the tunnel restarted.
