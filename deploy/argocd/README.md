# ArgoCD GitOps (Track B)

ArgoCD runs **inside** minikube and pulls from GitHub — see [ADR-0003](../../docs/decisions/0003-auto-deploy-argocd.md) for why a pull model is the only thing that works behind NAT.

## Prerequisites (do these first)

1. An image must exist in GHCR. It lands there after the first merge to `main` (CI pushes `:<sha>`) or the first release (`:v0.1.0`).
2. Make the GHCR package **public** so minikube can pull without a secret:
   `github.com/users/corleonesado/packages/container/pingapp/settings` → Change visibility → Public.
3. Because the repo is **private**, ArgoCD needs read access to fetch the chart. Add a read-only credential (fine-grained PAT with `Contents: read`, or a deploy key):
   ```bash
   argocd repo add https://github.com/corleonesado/pingapp.git \
     --username corleonesado --password <READ_ONLY_PAT>
   # or via kubectl: a repo Secret labelled argocd.argoproj.io/secret-type=repository
   ```

## Bootstrap

```bash
make argocd-install     # installs ArgoCD into the argocd namespace
make argocd-password    # prints the initial admin password
make argocd-ui          # port-forwards the ArgoCD UI to https://localhost:8081
make argocd-app TAG=v0.1.0   # applies the Application (image.tag override)
```

## Demonstrate the GitOps loop

1. Bump `image.tag` in `application.yaml` (or push a new chart change to `main`).
2. ArgoCD detects the change within its poll interval (~3 min, or click **Refresh**/**Sync** in the UI).
3. `kubectl get pods -n default -l app.kubernetes.io/instance=pingapp -w` shows the rollout.

`syncPolicy.automated` with `prune` + `selfHeal` means manual `kubectl edit`s are reverted to match Git.
