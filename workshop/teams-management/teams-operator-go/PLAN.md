# Plan: Golang Teams Operator

## Goal

Create a Golang drop-in replacement for the Python `teams-operator` that:
- Exposes the same environment variables (`TEAMS_API_URL`, `POLL_INTERVAL`, `LOG_LEVEL`)
- Produces identical Kubernetes namespaces with identical labels and annotations
- Follows the same reconciliation algorithm (poll-based, state-aware)
- Can be deployed by swapping the container image in the existing `operator-deployment.yaml`
- Meets the same security constraints (non-root, UID 1001, read-only root FS)

## Python Operator Behaviour Summary

The Python operator (`teams_operator.py`) does the following on each poll cycle:

1. **Fetch teams** from `GET /teams` on the Teams API
2. **Query the cluster** for namespaces labeled `app.kubernetes.io/managed-by=teams-operator` to discover which team IDs already have namespaces
3. **Set `known_teams`** to the cluster state (handles restarts gracefully)
4. **Diff**: `new_teams = api_teams - known_teams`, `deleted_teams = known_teams - api_teams`
5. **Create namespaces** for new teams with:
   - Name: `team-<sanitized-name>` (lowercase, hyphens for special chars, max 63 chars, prefixed)
   - Labels: `app.kubernetes.io/managed-by=teams-operator`, `teams.example.com/team-id=<id>`, `teams.example.com/team-name=<sanitized-name>`
   - Annotations: `teams.example.com/original-team-name=<name>`, `teams.example.com/created-by=teams-operator`, `teams.example.com/team-id=<id>`
6. **Delete namespaces** for removed teams
7. **Update known_teams** to the current API state
8. Sleep for `POLL_INTERVAL` seconds and repeat

## Go Operator Design

### Architecture

```
main.go              Entry point: config, signal handling, health server
  |
  +-- operator.go     TeamsOperator: reconcile loop, K8s client, Teams API client
```

Single `package main`, two source files. No sub-packages — mirrors the simplicity of the Python single-file operator.

### Key Design Decisions

| Concern | Python version | Go version | Rationale |
|---------|---------------|------------|-----------|
| K8s client | `kubernetes` Python client | `k8s.io/client-go` | Canonical Go client |
| HTTP client | `aiohttp` | `net/http` | Standard library sufficient for simple GET |
| Logging | `logging` module | `log/slog` | Structured logging, Go 1.21+ |
| Concurrency | `asyncio` | Goroutine + `time.Ticker` | Idiomatic Go |
| Health probe | exec: `python -c "import sys; sys.exit(0)"` | HTTP GET `/healthz` on port 8081 | More Kubernetes-native; avoids need for shell in container |
| Graceful shutdown | `KeyboardInterrupt` | `os.Signal` channel (SIGINT/SIGTERM) | Idiomatic Go |

### Environment Variables (identical to Python)

| Variable | Default | Description |
|----------|---------|-------------|
| `TEAMS_API_URL` | `http://teams-api-service:80` | Teams API base URL |
| `POLL_INTERVAL` | `30` | Reconciliation interval in seconds |
| `LOG_LEVEL` | `INFO` | Log level: DEBUG, INFO, WARN, ERROR |

### Namespace Sanitization

Must match the Python version character-for-character:
1. Lowercase the team name
2. Replace non-alphanumeric characters with hyphens
3. Collapse consecutive hyphens into one
4. Strip leading/trailing hyphens
5. Truncate to 63 chars, stripping trailing hyphens
6. Prefix with `team-`

### Drop-in Deployment Changes

Only two lines change in `operator-deployment.yaml`:

```yaml
# Before (Python):
image: olivercodes01/teams-operator:0.0.1

# After (Go):
image: olivercodes01/teams-operator-go:0.0.1
```

Health probes switch from exec to HTTP:

```yaml
# Before (Python):
livenessProbe:
  exec:
    command: ["python", "-c", "import sys; sys.exit(0)"]

# After (Go):
livenessProbe:
  httpGet:
    path: /healthz
    port: 8081
```

Everything else (RBAC, security context, env vars, resources) stays identical.

## File Listing

```
teams-operator-go/
  PLAN.md              This document
  main.go              Config parsing, health server, signal handling, main()
  operator.go          TeamsOperator struct, reconcile(), sanitizeNamespaceName()
  go.mod               Go module definition
  go.sum               Dependency checksums (generated)
  Dockerfile           Multi-stage build → non-root scratch/distroless image
  build.sh             Build & load into kind cluster
  deployment.yaml      Drop-in K8s manifests (same RBAC, updated image + probes)
```

## Dependency Graph

```
main.go
  ├── operator.go        (TeamsOperator, reconcile logic)
  ├── k8s.io/client-go   (Kubernetes API client)
  ├── k8s.io/apimachinery (API types)
  ├── net/http           (Teams API client)
  ├── log/slog           (structured logging)
  └── os/signal          (graceful shutdown)
```

## Quality Gates

Every code change must pass these checks before merging:

1. **`go vet ./...`** — static analysis for suspicious constructs (printf format mismatches, unreachable code, unused result values, lock copy issues, etc.)
2. **`go fix ./...`** — apply automated style and deprecation fixes, then re-check for any remaining warnings
3. **`go build ./...`** — zero warnings (unused imports, unused variables, shadowed variables)
4. **No unused constants or variables** — the compiler is strict; if a const/var exists, it must be referenced
5. **`govulncheck ./...`** — scan Go dependencies for known CVEs and vulnerability advisories before every release. Run `go install golang.org/x/vuln/cmd/govulncheck@latest` if not already available, then `govulncheck ./...`

If any of these produce output, fix it before proceeding.

## Vulnerability History

### Initial Scan (2026-05-03)

6 vulnerabilities found in transitive dependencies (none called by our code):

| ID | Module | Found | Fixed | Issue |
|---|---|---|---|---|
| GO-2026-4441 | `golang.org/x/net` | v0.23.0 | v0.45.0 | Infinite parsing loop |
| GO-2026-4440 | `golang.org/x/net` | v0.23.0 | v0.45.0 | Quadratic parsing complexity |
| GO-2025-3595 | `golang.org/x/net` | v0.23.0 | v0.38.0 | XSS in x/net |
| GO-2025-3503 | `golang.org/x/net` | v0.23.0 | v0.36.0 | HTTP Proxy bypass via IPv6 Zone IDs |
| GO-2025-3488 | `golang.org/x/oauth2` | v0.10.0 | v0.27.0 | Unexpected memory consumption in token parsing |
| GO-2024-3333 | `golang.org/x/net` | v0.23.0 | v0.33.0 | Non-linear case-insensitive parsing |

### Fix Applied (2026-05-03)

Upgraded transitive deps to resolve all 6 CVEs:

```
go get golang.org/x/net@v0.45.0 golang.org/x/oauth2@v0.27.0
go mod tidy
```

This also pulled in upgrades to related `golang.org/x` packages:

| Package | Before | After |
|---|---|---|
| `golang.org/x/net` | v0.23.0 | v0.45.0 |
| `golang.org/x/oauth2` | v0.10.0 | v0.27.0 |
| `golang.org/x/sys` | v0.18.0 | v0.36.0 |
| `golang.org/x/term` | v0.18.0 | v0.35.0 |
| `golang.org/x/text` | v0.14.0 | v0.29.0 |

Post-fix `govulncheck ./...` reports **No vulnerabilities found**.

### Vulncheck Fix Pattern

When `govulncheck` reports vulnerabilities in transitive dependencies:

1. Check whether your code actually calls the vulnerable symbols (`govulncheck -show verbose ./...`). If not, the risk is lower but upgrades are still recommended.
2. Identify the minimum fixed version from the `govulncheck` output.
3. Upgrade: `go get <module>@<fixed-version>` for each vulnerable module.
4. Run `go mod tidy` to reconcile the dependency graph.
5. Re-run quality gates: `go build ./...`, `go vet ./...`, `govulncheck ./...` — all must pass clean.
6. Document the upgrade in the Vulnerability History section of this file.

## Build & Deploy Sequence

1. `cd teams-operator-go`
2. `go mod tidy` — resolve dependencies
3. `go vet ./...` — static analysis (must be clean)
4. `go build -o teams-operator .` — verify compilation (zero warnings)
5. `govulncheck ./...` — check dependencies for known vulnerabilities
6. `./build.sh` — builds Docker image, loads into kind
7. `kubectl apply -f deployment.yaml` — deploy to cluster
8. `kubectl logs -f deployment/teams-operator-go -n engineering-platform` — verify