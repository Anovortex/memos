# Customized Memos Project Plan

## Audit scope and baseline

This plan records the audit of the upstream `usememos/memos` `main` branch before
product customization. No rebranding or architectural changes are included.
Repository-specific guidance in `AGENTS.md`, the root README, the file-server
README, build configuration, Compose file, and relevant source were reviewed.

The intended product is server-first: the responsive web client is a view over
server-owned data, not an offline replica. Quick notes, longer Markdown
documents, tags, search, attachments, dark mode, and responsive navigation
already have useful upstream foundations. Offline synchronization is not
required and should not be added.

## Current architecture

- **Backend:** a Go 1.26.2 application using Echo v5. `cmd/memos/main.go`
  configures the process; `server/server.go` wires HTTP routing, Connect RPC,
  gRPC-Gateway-compatible APIs, static frontend delivery, attachment delivery,
  and background runners. Public API definitions live in `proto/api/v1`.
- **Frontend:** a React 19 and TypeScript single-page application built by Vite
  8, with Tailwind CSS v4, React Router, React Query, and Connect Web. Server
  state is fetched through hooks; there is no client database or offline data
  synchronization layer.
- **Production packaging:** `pnpm release` places the built SPA in
  `server/router/frontend/dist`; Go embeds that directory into the server
  binary. `scripts/Dockerfile` produces a static Go binary in a multi-stage
  build and runs it in Alpine 3.21 as UID/GID 10001 after its entrypoint fixes
  volume ownership.
- **Development topology:** Vite listens on port 3001 and proxies `/api`,
  `/memos.api.v1`, `/file`, and the SSE route to the Go process on port 8081.
  The two processes must both run for the complete development UI.
- **Database:** a store facade and driver interface support SQLite, MySQL, and
  PostgreSQL. Each driver has fresh-install SQL and incremental migrations.
  SQLite is the default; production should use PostgreSQL.
- **Attachments:** attachment metadata is stored in the application database.
  Blobs may be stored in the database, on local server storage, or in an
  S3-compatible service. File responses, thumbnails, range requests, and S3
  presigned URLs are handled server-side.
- **Search and content:** notes are Markdown-native. Existing APIs and UI
  support tags, filters/search, attachments, and timeline-oriented quick
  capture. Longer-document UX should be evaluated as a customization rather
  than implemented during this baseline audit.

## Exact local development instructions

Prerequisites are Go 1.26.2, Node.js 24 or newer, and pnpm 11.0.1. Docker is
needed for the container-backed MySQL/PostgreSQL store tests and for testing the
production container topology. Buf is needed only when checking or changing
protobuf definitions.

From the repository root:

```bash
cd web
pnpm install
cd ..

# Terminal 1: SQLite backend, explicitly keeping mutable data out of the source root.
mkdir -p .local/memos
go run ./cmd/memos --port 8081 --data .local/memos

# Terminal 2: frontend development server and API proxy.
cd web
pnpm dev
```

Open `http://localhost:3001`. The backend health/API surface is on
`http://localhost:8081`. The audit verified HTTP 200 responses from both the Go
backend and the Vite frontend when they were run together. A source-built Go
backend by itself does **not** contain a freshly built SPA unless
`web/src` has first been released into the embedded frontend directory; this is
expected, so use Vite during development.

Useful verification commands are:

```bash
go test ./...
golangci-lint run

cd web
pnpm lint
pnpm test
pnpm build

cd ../proto
buf lint
buf format --diff
```

To verify an embedded production-style frontend before building Go:

```bash
cd web
pnpm release
cd ..
go run ./cmd/memos --port 8081 --data .local/memos
```

`pnpm release` changes generated embedded assets and should not be committed
unless a release workflow explicitly requires them.

## Audit verification results

| Check | Result |
| --- | --- |
| `cd web && pnpm lint` | Passed, including TypeScript checking and Biome. |
| `cd web && pnpm test` | Passed: 214 tests. |
| `cd web && pnpm build` | Passed. Vite emitted non-fatal large/chunk-size warnings. |
| `go test ./...` | Passed. Container-backed MySQL and PostgreSQL cases were skipped because the Docker daemon was unavailable. |
| `golangci-lint run` with v2.11.3 | Failed on seven existing epoch-naming issues; these predate and are unrelated to this docs-only change. |
| `buf lint` / generation checks | Not run because `buf` was unavailable. |
| `docker compose -f scripts/compose.yaml config` | Parsed successfully. |
| Docker runtime test | Not run because the Docker daemon was not running. |
| Backend and frontend startup | Both started successfully and returned HTTP 200. |

The repository does not provide `golangci-lint` locally, so v2.11.3 was
installed into a temporary directory to match CI. The first invocation was
blocked by the sandboxed default cache path:

```text
2026/07/16 19:49:13 failed to initialize build cache at /Users/asifkhan/Library/Caches/golangci-lint: mkdir /Users/asifkhan/Library/Caches/golangci-lint: operation not permitted
```

After redirecting that cache to `/tmp`, the pinned linter ran and returned:

```text
server/router/api/v1/attachment_service.go:290:2: epoch-naming: var currentTs should have one of these suffixes: Sec, Second, Seconds (revive)
server/router/api/v1/memo_service.go:474:4: epoch-naming: var updatedTs should have one of these suffixes: Sec, Second, Seconds (revive)
server/router/api/v1/memo_update_helpers.go:17:2: epoch-naming: var updatedTs should have one of these suffixes: Sec, Second, Seconds (revive)
server/router/api/v1/test/memo_share_service_test.go:192:2: epoch-naming: var expiredTs should have one of these suffixes: Sec, Second, Seconds (revive)
server/router/api/v1/user_service.go:309:2: epoch-naming: var currentTs should have one of these suffixes: Sec, Second, Seconds (revive)
store/test/attachment_filter_test.go:170:2: epoch-naming: var now should have one of these suffixes: Sec, Second, Seconds (revive)
store/test/memo_filter_test.go:636:2: epoch-naming: var now should have one of these suffixes: Sec, Second, Seconds (revive)
7 issues:
* revive: 7
```

These findings predate and are unrelated to this documentation-only change;
address them separately rather than mixing cleanup into the baseline plan.

The missing Buf dependency was reported as:

```text
zsh:1: command not found: buf
```

The Docker client and Compose plugin were installed, but the daemon check
failed with:

```text
Cannot connect to the Docker daemon at unix:///Users/asifkhan/.docker/run/docker.sock. Is the docker daemon running?
```

## Production deployment architecture

The checked-in `scripts/compose.yaml` is a valid single-container SQLite
example, not the target production architecture. Production should use a
project-owned Compose overlay/file that preserves the upstream service while
adding PostgreSQL:

1. A `memos` service built from the audited commit (or a versioned internal
   image), listening internally on 5230.
2. A supported PostgreSQL service on a pinned major version with a named,
   server-controlled data volume.
3. `MEMOS_DRIVER=postgres` and `MEMOS_DSN` supplied through a Compose secret or
   `MEMOS_DSN_FILE`, not committed in plaintext.
4. A named Memos data volume mounted at `/var/opt/memos`, even with PostgreSQL,
   because local attachments and other server-owned files may live there.
5. A TLS reverse proxy in front of Memos, with a stable hostname, upload limits,
   security headers, and WebSocket/SSE-safe proxy settings. Do not publish
   PostgreSQL to the public network.
6. Health checks, restart policies, resource limits, log rotation, and
   monitoring. Deploy only immutable tagged images; never depend on `stable`
   silently changing.

An illustrative DSN shape is
`postgres://memos:PASSWORD@postgres:5432/memos?sslmode=disable` inside an
isolated Compose network. `sslmode=disable` is acceptable only for that private
container network; external PostgreSQL requires verified TLS. Exact secret,
proxy, and backup choices should be committed in the deployment phase after
the target host is known.

## Data and attachment storage paths

- With local development defaults, SQLite is `memos_prod.db` under the
  resolved `--data` directory. The instructions above therefore create
  `.local/memos/memos_prod.db`.
- In the official container, the application data directory is
  `/var/opt/memos`, backed by the host/volume mounted there. The upstream
  example maps `~/.memos/` to that directory.
- PostgreSQL stores notes, users, settings, attachment metadata, and
  database-backed attachment blobs in its own PostgreSQL data volume. `MEMOS_DSN`
  identifies that database.
- New installations default to local attachment storage. The default template
  is `assets/{timestamp}_{uuid}_{filename}`, resolved beneath the Memos data
  directory, so container-local attachments normally land below
  `/var/opt/memos/assets/`. The stored reference may also be absolute if an
  administrator deliberately configures an absolute path.
- Database attachment storage keeps the blob in the attachment row and grows
  the PostgreSQL backup accordingly. S3 storage keeps object data in the
  configured bucket and metadata/configuration in PostgreSQL.

For the private-server requirement, allow only server-controlled local storage,
the production database, or a self-hosted S3-compatible target. Disable or
operationally forbid arbitrary external attachment links if they conflict with
the policy that all application data remain on the server.

## PWA implementation status

The frontend is partially PWA-ready:

- `web/public/site.webmanifest` is linked from `web/index.html`.
- It defines a name, short name, 192px and 512px icons, `start_url`, root scope,
  theme/background colors, and `display: "standalone"`.
- iOS standalone and status-bar metadata are present, and theme color is
  updated with the active light/dark theme.
- The responsive SPA already supports mobile and desktop layouts.

There is no registered service worker or offline cache/synchronization system.
That matches the server-first data requirement but installability and platform
behavior still require a real-device acceptance pass on iOS, Android, and
macOS. In the PWA phase, validate HTTPS installation prompts, icons/splash
screens, safe areas, viewport behavior, share/file handling if desired, and
navigation after an app update. Any service worker added should be a minimal
app-shell/update mechanism and must not create an offline note store.

## Authentication status

Memos supports local username/password sign-in and configurable OAuth2 identity
providers. Authentication uses short-lived HS256 JWT access tokens (15
minutes), long-lived refresh tokens (30 days), and personal access tokens for
API clients. Refresh tokens are maintained in server-side records and delivered
using the refresh-token cookie; access tokens are sent as bearer tokens.
Cross-tab browser auth state is coordinated with `BroadcastChannel`.

An unset instance URL produces private access behavior; setting one enables
anonymous/public access according to the current profile logic. Before
production, explicitly test private-mode ACLs, bootstrap-owner creation,
sign-in/sign-out, refresh rotation/revocation, password reset/recovery,
OAuth failure recovery, PAT permissions, cookie `Secure`/`SameSite` behavior
behind the chosen proxy, and rate limiting. Keep password login enabled until a
tested administrative recovery route exists. TLS is mandatory.

## Backup requirements

Backups must be server-side, encrypted, monitored, and regularly restored in a
separate environment:

- Take PostgreSQL-native logical backups (`pg_dump` in custom format) on a
  schedule, plus provider/volume snapshots or WAL-based point-in-time recovery
  when the recovery-point objective requires it.
- Back up the complete Memos data volume, especially `assets/` and any custom
  local attachment path. Preserve ownership and permissions.
- If attachments use self-hosted S3, enable bucket versioning and back up or
  replicate object data independently. A database dump alone is insufficient.
- Back up deployment configuration, Compose files, proxy configuration, and
  secret references. Store actual secrets in the approved secret manager, with
  a separately protected recovery procedure.
- Coordinate database and attachment snapshots closely enough to avoid
  dangling metadata or missing objects. Record retention, off-host copies,
  encryption keys, recovery-point objective, and recovery-time objective.
- Automate backup success alerts and perform documented restore drills. A
  backup is not accepted until a restore has been verified.

For SQLite development, stop writes or use SQLite's online backup mechanism;
copying a live database file naively can yield an inconsistent backup.

## Recommended customization phases

1. **Baseline and governance:** pin the audited upstream commit, document the
   upstream remote and update process, make CI reproduce frontend, Go, proto,
   and Compose checks, and resolve or explicitly baseline existing lint debt.
2. **Production foundation:** add the project-owned PostgreSQL Compose
   deployment, secrets, TLS proxy, health checks, server-controlled attachment
   policy, monitoring, and tested backup/restore automation.
3. **PWA and responsive acceptance:** test installability and navigation on
   iOS, Android, and macOS; fix safe-area, keyboard, upload, editor, and
   responsive-navigation issues. Add only minimal non-data service-worker
   behavior if installation quality requires it.
4. **Privacy and security hardening:** default to private access, audit ACLs and
   external URLs, validate authentication recovery and session behavior, add
   rate limits/security headers, and document operational access.
5. **Core product fit:** refine quick capture and longer-document editing while
   preserving tags, search, attachments, dark mode, and existing features.
   Establish browser and mobile regression tests before visual changes.
6. **Branding and design:** only after the preceding acceptance gates, apply
   product identity and a coherent responsive design without replacing stable
   upstream primitives unnecessarily.
7. **Operations and upgrades:** rehearse upgrades, migrations, rollback, and
   restores; publish a release checklist and support matrix.

## Risks of maintaining an upstream fork

- Long-lived changes to shared UI, generated API files, migrations, auth, or
  Docker workflows create recurring merge conflicts.
- Upstream may change Go/Node/pnpm versions, database migrations, API contracts,
  or attachment semantics. Skipped releases increase migration and security
  risk.
- Editing generated protobuf output instead of source definitions makes the
  fork difficult to regenerate and review.
- A PostgreSQL-only customization can accidentally break upstream's SQLite and
  MySQL assumptions; schema changes must continue to cover all three drivers
  unless support is deliberately changed in a separately approved decision.
- Local attachment paths and database/S3 transitions can create incomplete
  backups or orphaned files during upgrades.
- Upstream's public/private and authentication behavior may evolve, so privacy
  requirements need automated regression tests rather than configuration
  assumptions.
- Committing built frontend assets creates noisy conflicts. Prefer reproducible
  release builds in CI.
- Tracking `stable` rather than immutable versions makes production behavior
  drift without code review.

Keep `origin` for the product fork and an `upstream` remote for
`usememos/memos`. Regularly fetch upstream, review release notes and migrations,
merge or rebase in small increments on a dedicated update branch, run the full
matrix, inspect the diff, test backup/restore and rollback, and only then
promote an immutable image. Prefer narrow adapters, configuration, and additive
components over rewriting upstream code.
