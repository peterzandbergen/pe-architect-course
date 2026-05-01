# AGENTS.md

This repo is the companion workshop for the PlatformEngineering.org Architect course. It is **a set of guided tutorials, not a buildable application**. There is no root-level package, no test suite, no CI, and no lint/typecheck pipeline. Most "work" is `kubectl` / `helm` / `terraform` commands that a student runs against a local kind cluster. Treat the repo accordingly.

## Layout

- `workshop/foundation/` — kind cluster verification, Grafana stack, OPA Gatekeeper, metrics-server. **Module 1, must run first.**
- `workshop/capoc/{cve,quality}/` — Gatekeeper constraint templates + sample deployments.
- `workshop/secops/` — Falco rules and constraints.
- `workshop/teams-management/` — the only real code:
  - `teams-api/` — FastAPI (`main.py`), in-memory store, listens on `:8000`, Service exposes `:4200`.
  - `teams-app/` — Angular 16 SPA, dev server on `:4200`, proxies API via `proxy.conf.json`. Uses Keycloak.
  - `teams-operator/` — kopf-based Python operator (`teams_operator.py`) that reconciles team namespaces from the Teams API.
  - `cli/teams_cli.py` — Python CLI client.
  - `keycloak/` — Keycloak deployment manifest.
- `workshop/capstone/` — final exercise (README only).
- `setup/kind/cluster.yaml` — kind config; cluster name is **`5min-idp`**, control-plane maps host 80/443/6443 to NodePorts 30080/30443/6443.
- `setup/terraform/` — Terraform 1.5.7 module that installs base IDP + ingress into the cluster. Always invoke with `terraform -chdir=setup/terraform <cmd>`.
- `.devcontainer/` — Coder/devcontainer bootstrap. `postCreateCommand.sh` is the source of truth for required tooling and cluster provisioning.

## Environment assumptions

The default runtime is a Coder/devcontainer workspace, not a local laptop. From `postCreateCommand.sh`:

- Kind cluster `5min-idp` is created at `$HOME/state/kube/config.yaml` (typically `/home/vscode/state/kube/config.yaml`). An internal kubeconfig (`config-internal.yaml`) is also exported for in-docker access.
- Local container registry runs at **`localhost:5001`** (containerd is configured to alias it on the kind nodes).
- `/etc/hosts` gets `127.0.0.1 5min-idp-control-plane`.
- Tools auto-installed if missing: `kubectl`, `kind` (v0.26.0), `helm`, `terraform` (1.5.7), `yq` (v4.35.1), `score-k8s` (0.1.18), `mkcert` (with `CAROOT=/workspaces`), `glow`, `jq`, `bash-completion`.
- `postStartCommand.sh` adds aliases `k`, `kg`, `h` (humctl), `sk` (score-k8s) and kubectl completion to `~/.bashrc`. These may need to be re-sourced after a workspace restart.
- Terraform reads `TF_VAR_tls_cert_string=$PIDP_CERT`, `TF_VAR_tls_key_string=$PIDP_KEY`, and `TF_VAR_kubeconfig=$kubeconfig_docker` — these env vars must be set before running terraform manually.

## Service access patterns (frequently confused)

| Where you are | URL pattern | Example |
|---|---|---|
| macOS/Windows + Coder Desktop VPN | `http://<workspace>.coder:<port>` | `http://supersandbox.coder:3000` |
| Through cluster ingress | `http://<svc>.127.0.0.1.sslip.io` | `http://teams-api.127.0.0.1.sslip.io` |
| Linux or fallback | `coder port-forward <ws> --tcp <local>:<svc>` then `http://localhost:<local>` | `http://localhost:3000` |

Coder Desktop is **not available on Linux** — use `coder port-forward` and substitute `localhost` everywhere READMEs say `<workspace-name>.coder`.

## Gotchas an agent will hit

- The foundation module deploys a Gatekeeper constraint `ns-must-have-gk` that blocks namespace creation without an `admission` label. **It must be deleted before later modules** (Falco, etc.) or their namespace creation will fail. See `workshop/foundation/README.md` "Remove the constraint after testing".
- Default Grafana login is `admin` / `admin123` (set in the values YAML the student creates).
- Teams API has no persistence — `teams_store` is an in-memory dict; restarting the pod wipes data. Don't add migrations or DB code unless the task explicitly asks.
- Teams API container port is `8000`, but the K8s Service port is `4200` (matches the UI dev server port). Port-forwarding examples in module READMEs use `4200:4200` against the Service.
- Angular app is v16 with `zone.js` 0.13 and TypeScript 5.1 — do not bump to standalone components / signals patterns without checking; the codebase uses NgModules (`app.module.ts`).
- The Teams operator is kopf-based Python, not a Go/kubebuilder operator. CRDs and reconciliation logic live in `teams_operator.py`.
- Many "deployments" in the workshop are intentionally broken (e.g. `deployment.yaml` vs `deployment-working.yaml` in `capoc/*` and `secops/`) to demonstrate policy denials. Don't "fix" the broken ones unless asked.
- `simple-constraint*.yaml` files referenced from the foundation README live at `workshop/foundation/`, not the repo root.

## Working on changes

- No formatter/linter is configured. Match the surrounding style of each subproject.
- There are no automated tests. To verify changes, the only realistic option is `kubectl apply`/`helm`/`terraform apply` against the local kind cluster, or `python -m uvicorn main:app` for the API and `npm run dev` for the UI.
- For the Angular app, dev work uses `npm run dev` (proxy + 0.0.0.0 host); production build is `npm run build:prod`. `npm test` runs Karma but no specs are wired up by default.
- For the Teams API: `pip install -r requirements.txt && uvicorn main:app --reload`.
- When editing tutorial markdown, preserve module ordering (foundation → capoc → secops → teams-management → capstone) and the `<workspace-name>.coder` / sslip.io / localhost URL patterns above.

## Authoritative references

- `workshop/README.md` — module overview, completion checklists, network access table.
- `workshop/TROUBLESHOOTING.md` — known issues; check before debugging.
- `workshop/foundation/README.md` — canonical setup walkthrough.
- `.devcontainer/postCreateCommand.sh` — ground truth for tool versions and cluster bootstrap.
