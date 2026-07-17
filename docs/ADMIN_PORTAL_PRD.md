# Noviledger Notes Owner Portal — Product Requirements Document

**Status:** Reviewed draft for owner approval  
**Product name:** Noviledger Notes  
**Production URL:** notes.noviledger.com  
**Audience:** Platform owner, engineering, operations  
**Development authorization:** Not approved until the owner explicitly approves this PRD

## 1. Product summary

Build a dedicated, responsive **Noviledger Notes** owner portal at `/admin` for operating the hosted note service. The portal is for the owner of the whole hosted platform, not for individual note-account owners. It is separate from the note-taking interface and gives the platform owner a CMS/CRM-style view of users, adoption, content activity, and storage consumption without access to anything users wrote or uploaded.

The portal is an operational surface, not a user-facing notes feature. Only the instance owner may access it. Frontend route guards improve navigation, while every portal API remains owner-authorized on the server.

## 2. Problem

The existing Memos administration is distributed across Settings and does not provide a clear operator view. The owner cannot quickly answer:

- How many users have access?
- Which accounts are active, archived, administrators, or regular users?
- How many notes and attachments has each user created?
- How much attachment storage does each user consume?
- When did a user last change note content?
- How much database and local attachment storage does the instance use?

The current owner-only Resources section partially exposes these metrics but is not the dedicated administrative product requested.

## 3. Goals

1. Provide a dedicated `/admin` portal with a distinct operator experience.
2. Show instance-wide usage totals and per-user usage in one place.
3. Give the owner efficient read-only workflows for finding and evaluating user accounts in Phase 1A.
4. Work cleanly on mobile and desktop.
5. Enforce owner-only access at the backend trust boundary.
6. Reuse existing Memos APIs, components, and conventions to minimize upstream-fork maintenance.

## 4. Non-goals for Phase 1

- Exact login history, session history, IP addresses, devices, or user-agent tracking.
- Reading, editing, searching, previewing, exporting, or otherwise retrieving anything users wrote or uploaded, including note text, rendered content, titles, tags, links, attachment filenames, attachment contents, and derived previews.
- Billing, subscriptions, quotas, invoices, or chargeback.
- Organization/team hierarchy, leads, sales pipelines, or general-purpose CRM records.
- Real-time analytics, charts requiring a separate analytics database, or long-term metric warehousing.
- Replacing the existing Memos authentication system.
- Allowing secondary administrators to access owner analytics.

## 5. Actors and authorization

### Instance owner

The only Phase 1 portal actor. The owner can view the portal and inspect instance/user usage. Account-management actions remain in Settings during Phase 1A and require separate Phase 1B approval.

The portal owner is identified by a persisted instance-owner user ID. On upgrade, the value is backfilled once from the lowest-ID current administrator. It never changes automatically when roles or users change.

The owner cannot be archived, demoted, or deleted through portal or general user-management APIs. Ownership transfer is outside Phase 1 and requires an explicit, audited recovery/transfer workflow. If the configured owner is missing, the portal remains locked, normal note-taking continues, and recovery requires a host-side administrative command that names an existing administrator. The system must never silently select a replacement owner.

### Secondary administrator

May use existing Memos administration features but cannot view `/admin` data or call owner portal APIs.

### Regular user and anonymous visitor

Cannot view the route or call owner portal APIs.

### Authorization requirements

- Unauthenticated requests return `Unauthenticated`/HTTP 401.
- Authenticated non-owner requests return `PermissionDenied`/HTTP 403.
- Authorization is checked before reading or returning cached portal data.
- UI route hiding is never treated as the security boundary.
- Portal APIs must not return passwords, tokens, secrets, attachment contents, or note contents.

## 6. Phase 1 user experience

### 6.1 Route and layout

- Canonical route: `/admin`.
- Dedicated admin layout, without the memo explorer/editor navigation.
- Header displays **Noviledger Notes**, the signed-in owner identity, refresh action, and “Back to notes”.
- Desktop: compact navigation and multi-column summary cards.
- Mobile: single-column cards, compact navigation, and user rows that remain readable without horizontal page overflow.
- Loading, empty, partial-data, permission-denied, and retry states must be explicit.

### 6.2 Overview

Display:

- Total users.
- Enabled accounts (`NORMAL` account state).
- Archived users.
- Administrator and regular-user counts.
- Total notes.
- Total attachments.
- Total attachment bytes.
- PostgreSQL database size.
- Memos local-data size.
- Timestamp of the generated statistics snapshot.

Metrics may be cached for up to 60 seconds. The portal must show when the snapshot was generated. Manual refresh re-fetches the snapshot but may return the cached value until its generated timestamp advances; Phase 1 does not add a cache-bypass operation.

### 6.3 User directory

For every user, display:

- Avatar, display name, and username.
- Role: owner/admin/user.
- Account state: enabled or archived.
- Account creation date.
- Note count, excluding comments.
- Attachment count.
- Attachment storage in human-readable units.
- Latest note-content activity, explicitly labeled as content activity rather than login activity.

The directory uses server pagination and supports server-side search by username/display name, filters for role and account state, and sorting by username, attachment storage, note count, and latest content activity. The API defines a maximum page size and stable page tokens/order.

Selecting a row opens a detail view without exposing note or attachment content. The owner can inspect the allowed account metadata, copy the username, and return to the same filtered/sorted page.

Primary Phase 1A operator journeys are:

1. Find a user by username or display name.
2. Identify accounts consuming the most logical attachment storage.
3. Identify accounts with no notes or old content activity without calling them “inactive users”.
4. Distinguish owner, secondary administrator, regular user, enabled account, and archived account.
5. Open a user detail view and return without losing search, filter, sort, or page state.

CSV export, bulk actions, internal contact notes, messaging, arbitrary CRM fields, and deep links into private user content are excluded from Phase 1A.

### 6.4 User management — Phase 1B

Phase 1A is read-only. Existing **Settings → Members** remains the account-management surface until Phase 1B is separately approved and accepted. Phase 1B reuses existing Memos account-management behavior:

- Create a user.
- Edit display name, email, role, and other already-supported account fields.
- Archive and restore a user.
- Delete an eligible archived user only through the existing confirmation and backend rules.

Destructive actions require an explicit confirmation. The owner cannot accidentally archive, demote, or delete the currently signed-in owner account through this portal.

### 6.5 Existing Settings pages

Do not remove **Settings → Resources** or **Settings → Members** in the first portal deployment. Remove the duplicate Resources entry only in a later release after `/admin` is production-verified and rollback-tested. Members remains until Phase 1B provides accepted replacement operations.

## 7. Data definitions

- **Lifetime note count:** `NORMAL` and `ARCHIVED` non-comment memos owned by the user. Deleted rows are absent and excluded.
- **Attachment count:** every extant attachment record owned by the user, including attachments associated with archived memos. Deleted attachments are excluded.
- **Attachment bytes:** sum of server-recorded attachment sizes, independent of local/database/S3 storage backend.
- **Latest content activity:** newest memo update timestamp across the same `NORMAL` and `ARCHIVED` non-comment set. It is not last login, last page view, or last token refresh.
- **Database size:** database-driver-reported database size.
- **Local-data size:** recursive size of the Memos data directory, when available.

Instance total notes and attachments are sums of these exact per-user definitions. Comments are not counted in Phase 1. The portal must show “Unknown” rather than zero when the backend cannot calculate a size.

Database size and local-data size are separate measurements and must never be summed or labeled as total storage. On the PostgreSQL deployment, database size describes PostgreSQL-managed data; local-data size includes server files and caches below the Memos data directory. Attachment bytes are logical uploaded bytes from attachment metadata, not measured disk allocation, and may represent local, database, or S3-backed objects. Display sizes use IEC units (KiB, MiB, GiB, TiB) with explanatory tooltips.

## 8. Functional requirements

| ID | Requirement |
| --- | --- |
| AP-001 | Only the instance owner can open `/admin`. |
| AP-002 | Only the instance owner can retrieve portal usage data through the API. |
| AP-003 | The overview displays all metrics defined in section 6.2. |
| AP-004 | The user directory displays all fields defined in section 6.3. |
| AP-005 | Server-side search, filtering, sorting, and pagination update the visible directory without mutating server data. |
| AP-006 | Phase 1A is read-only and does not expose account mutations. |
| AP-007 | Manual refresh re-fetches the snapshot and clearly retains the generated timestamp when the server cache has not expired. |
| AP-008 | The portal is usable at 320px viewport width and on current desktop browsers. |
| AP-009 | Secondary admins, regular users, and anonymous users cannot retrieve portal data by calling APIs directly. |
| AP-010 | No note body, attachment blob, credential, token, or secret is included in portal responses. |
| AP-011 | Platform-owner status does not permit reading, searching, exporting, previewing, or downloading another user's private note or attachment content through any application or API route. |

## 9. Technical capability contract

- Add a lazy-loaded owner-guarded `/admin` route under the existing authenticated root layout.
- Add a small responsive admin layout and portal page; do not introduce a new frontend framework or state system.
- Persist an immutable instance-owner user ID, backfill it once, protect that user from deletion/demotion/archive, and provide an explicit host-side recovery command. Fresh installs and supported upgrades must behave consistently across SQLite, PostgreSQL, and MySQL.
- Add an owner-only, server-paginated directory API backed by database aggregation. It must use `COUNT`, `SUM`, and `MAX` grouped by creator/user and must not materialize note or attachment rows in application memory.
- The portal directory DTO allowlist is: user resource name, username, display name, avatar URL, derived portal role, account state, account creation time, lifetime note count, attachment count, logical attachment bytes, and latest content activity. Email is excluded from list responses and may be fetched only by an owner-authorized detail/management operation.
- Preserve protobuf wire compatibility through additive fields only.
- Keep server owner checks centralized through the existing owner resolver.
- Continue using React Query and Connect RPC clients.
- Reuse existing avatar, badge, table, dialog, and user-management components where their current contracts fit.
- Keep the 60-second overview cache for Phase 1 and return a generated timestamp.
- Return section availability/error metadata so an unavailable user directory cannot be mistaken for zero users. Internal error details are logged server-side and not returned.
- Owner metadata responses must be private and must not be persisted by a service worker or shared cache.

## 10. Privacy and security

- The portal reports account metadata and aggregate usage only. The platform owner can see that a user has a given number of notes or attachment bytes, but cannot see what any note or attachment contains.
- User email is excluded from overview/list payloads and may appear only in an owner-authorized detail or existing member-management request.
- No note content or attachment content is exposed.
- Portal responses contain no note text, titles, tags, links, attachment filenames, attachment references, thumbnails, previews, search snippets, or content-derived summaries.
- The owner role grants operational metadata access only and must not grant access to another user's private notes or attachments through existing application pages, search, exports, or direct APIs. Phase 1 acceptance includes an audit and denial tests for those existing paths.
- No IP address, device, or user-agent collection is introduced in Phase 1.
- Errors shown to the owner must not expose database queries, filesystem paths, or secrets.
- Every new read or mutation must have owner/non-owner tests.

## 11. Reliability and performance

- A portal failure must not prevent normal note-taking.
- Missing database-size or filesystem-size information must degrade to “Unknown”; other metrics should still render.
- Phase 1 target: overview and a directory page complete within two seconds against seeded 1,000-user, 100,000-note, and 100,000-attachment fixtures on production-like PostgreSQL.
- Usage is aggregated in the database and user results are server-paginated with a maximum page size of 100.
- A directory/usage failure is reported for that section and never rendered as zero users. Other available overview metrics may continue rendering.

## 12. Acceptance criteria

1. The owner can visit `/admin` directly and through an owner-only navigation link.
2. Secondary admins, regular users, and anonymous visitors are denied by both route and API.
3. Overview totals match seeded database fixtures.
4. Per-user note, attachment, and byte totals match seeded fixtures, including more than 100 attachments.
5. Latest content activity is correctly labeled and never presented as login activity.
6. Server-side user search, role/state filters, sorting, stable pagination, and return-to-filtered-list behavior work on desktop and mobile.
7. Phase 1A exposes no account mutation controls; existing Settings member-management behavior remains unchanged.
8. The persisted owner cannot be archived, demoted, or deleted by any existing user-management API.
9. Loading, empty, unknown-size, API-error, and permission-denied states are tested.
10. Frontend tests, type checking, linting, production build, focused Go tests, protobuf lint/format, and Docker build pass.
11. Production deployment preserves `/srv/memos-data/postgres` for PostgreSQL and `/srv/memos-data/memos` for Memos files; portal work does not mutate storage mounts.
12. Public application health returns HTTP 200 after deployment.
13. The host-side recovery command rejects nonexistent or non-admin targets, updates owner identity atomically, emits an audit record without secrets, and restores portal access only to the named administrator. Failure leaves the previous owner value unchanged. Missing-owner lockout and successful/failed recovery are tested across supported databases.
14. Privacy tests prove that the platform owner cannot retrieve another user's private note text, tags, search results, exports, attachment filename, attachment blob, thumbnail, preview, or download through existing UI and API paths.

### Phase 1B acceptance gate

Phase 1B is not authorized by approval of Phase 1A. When separately approved, it must verify onboarding, editing, archive, restore, delete eligibility/failure handling, owner self-protection, and rollback while preserving all existing server validations.

## 13. Phase 2: authentication audit

Phase 2 may add true access reporting:

- Last successful login timestamp.
- Append-only successful login events.
- Authentication method such as password or SSO.
- Session/token revocation events and retention policy.

IP address and user-agent collection are excluded by default because they add privacy and security obligations. They require a separate owner decision, retention period, disclosure policy, and threat review.

Phase 2 must use database migrations for SQLite, PostgreSQL, and MySQL and must not infer login activity from note updates or personal-access-token usage.

## 14. Rollout and rollback

1. Implement through RED/GREEN tests and additive API changes.
2. Deploy as a new immutable Docker image.
3. Verify owner access and non-owner denial before announcing availability.
4. Verify database and attachment mounts still point to `/srv/memos-data`.
5. Retain the previous image for immediate rollback.
6. The portal requires no destructive data migration in Phase 1.

Owner persistence may require additive instance-setting/schema migration work, but it must not rewrite note or attachment data. A pre-deployment database backup and tested rollback are required before that migration is applied.

## 15. Open decisions for owner approval

1. **Login audit:** approve or reject a separate Phase 2 without IP/device collection initially.
2. **Phase 1B timing:** decide after Phase 1A production acceptance whether user management should move from Settings into the portal.
3. **Ownership transfer:** define a future audited owner-transfer workflow; Phase 1 supports host-side recovery only.

## 16. Delivery phases

### Phase 1A — Owner portal foundation

Persisted owner model and recovery, `/admin` route, owner guard, layout, SQL-aggregated overview, paginated user directory/detail, responsive behavior, and production verification. Existing Settings navigation remains during rollout.

### Phase 1B — Account operations

Integrate existing create/edit/archive/restore/delete workflows and protect the owner account.

### Phase 2 — Authentication audit

Persist and display true login events after separate privacy and retention approval.

## 17. Upstream-fork boundaries

- Expected touched surfaces are owner persistence and supported-driver migrations/fresh schemas; shared owner resolution and mutation guards; additive protobuf plus generated Go/TypeScript/OpenAPI; database usage aggregation; owner-only route/guard/layout/page/navigation; i18n; and tests.
- Generated protobuf/OpenAPI files are regenerated through Buf and never edited manually.
- Owner enforcement must preserve upstream role and administrator behavior outside the portal and explicit owner-protection rules.
- Upstream merge verification must cover first-user setup, admin promotion/demotion, owner deletion prevention, profile owner identity, missing-owner recovery, database migrations, portal denial, and rollback.
- Existing Settings surfaces are removed only after their replacement is accepted and independently recoverable.

## 18. Handoff gate

Development must not begin until the owner:

1. Approves this PRD.
2. Resolves or accepts the recommended defaults in section 15.
3. Authorizes Claude to implement Phase 1A, followed by review and deployment through the established workflow.
