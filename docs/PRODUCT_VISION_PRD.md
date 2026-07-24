# Noviledger Notes — Product Vision & Master PRD

**Status:** Approved (with tweaks) by owner — 2026-07-25
**Working product name:** TBD — placeholder only. ("Noviledger" is a separate product; name is decided in Stage 3.)
**Base:** Fork of [usememos/memos](https://github.com/usememos/memos)
**Delivery model:** Hosted, publicly available SaaS (multi-user, per-user isolation)
**Monetization:** Free to use in v1, but free-tier caps (note count / AI tokens) are designed in from day one so a paid tier drops in later without rework. Billing implementation deferred.
**Development authorization:** Approved — Stage 2 (AI foundation + hardening) may begin.

---

## 1. Vision

Turn the memos note engine into a distinct, publicly usable **AI-native notes product** — the way
Cursor turned VSCode into an AI-native editor. memos is the proven core (markdown notes, tags,
attachments, per-user accounts, self-hostable, fast). The product's reason to exist is a first-class
**AI layer**: your notes become something you *talk to, search by meaning, and get organized for you*,
not just a place you type into.

The bet: general note apps (Notion, Obsidian, Apple Notes) bolt AI on as a side feature. This product
makes AI the primary interface to your own knowledge.

## 2. Positioning

| Product | Core loop | AI |
|---------|-----------|----|
| memos (base) | Fast personal microblog/notes | None |
| Notion / Obsidian | Structured docs / linked notes | Bolt-on assistant |
| **Noviledger Notes** | Capture fast, **ask/organize with AI** | **Native, the point of the product** |

One-line pitch: *"Your notes, but you can ask them anything."*

## 3. What we keep from memos (do NOT rebuild)

- Note (memo) storage, markdown rendering, tags, attachments/resources.
- Per-user accounts, roles (admin/user), auth + session/refresh-token system.
- Per-memo visibility (private / protected / public).
- Self-host packaging, DB layer (SQLite/Postgres/MySQL), proto/Connect API.
- The owner operations portal (already specced in `ADMIN_PORTAL_PRD.md`).

Reusing this base is the entire strategic advantage. Every feature below must extend the existing
API/schema/component conventions, not replace them, to keep the fork maintainable against upstream.

## 4. The multi-tenancy reality (important — scopes the whole project)

memos is **single-instance, per-user** with no organization/workspace/team layer.

For an **AI-native personal-notes SaaS**, that is *already enough*:
- "Public instance" mode (set `InstanceURL`, allow registration) gives open self-service signup.
- Existing per-user data isolation gives every signup their own private notes.

Therefore **we do NOT build orgs/workspaces/team tenancy in v1.** That layer is only required if/when
collaboration becomes a feature — which is explicitly *not* the differentiator. Building it now is
speculative work (YAGNI). Stage 3 ("multi-tenancy architecture") is scoped to **hardening the existing
public multi-user substrate for production**, not to adding an org model.

Deferred to a future phase, only if teams ship: workspaces, shared spaces, seat-based membership.

**Trigger to revisit (explicit):** build the org layer only when at least one is true — (a) paying users
ask to share a workspace with named teammates, (b) a paid *team* plan is on the roadmap, or (c) collaborative
editing becomes a committed feature. Absent one of these, orgs stay out of scope. Per-user isolation +
public-instance signup remain the substrate.

## 5. Differentiator: the AI layer (the actual net-new product)

This is where nearly all new engineering goes. Delivered as capabilities on top of the existing memo store.

### 5.1 Capabilities (priority order)
1. **Semantic search** — find notes by meaning, not keyword. Embeddings over each user's memos.
2. **Ask your notes (RAG chat)** — natural-language Q&A grounded in the user's own notes, with citations
   back to source memos.
3. **AI organize** — auto-suggest tags, summaries, and links between related notes.
4. **AI writing assist** — expand, rewrite, summarize a memo in the editor.

### 5.2 Architecture (extends memos, does not replace)
- **Embedding store:** pgvector (Postgres) as the production target; keep SQLite path working for
  self-host via a pluggable vector interface. Reuse the repository pattern already in `store/`.
- **Indexing:** background worker embeds memos on create/update. Per-user scoped; never cross-user.
- **LLM/embedding provider:** pluggable provider interface (start with one provider, e.g. Claude +
  an embedding model). Provider keys are server-side env config, never client-exposed.
- **RAG:** retrieve top-k user-scoped chunks → prompt with citations → stream answer to client.
- **Isolation (CRITICAL):** every AI query is hard-scoped to the requesting user's data at the trust
  boundary, mirroring existing memo visibility rules. AI must never surface another user's notes.
- **Cost/abuse controls:** per-user rate limits and daily token caps from day one (even while free),
  so a free tier can't bankrupt the instance.

### 5.3 New surfaces
- Search bar upgraded to semantic + keyword hybrid.
- A "Chat with your notes" panel.
- Inline editor AI actions.

### 5.4 Free-tier limits (freemium-ready from day one)
The product is free to use in v1, but limits are designed in now so a paid tier is a config change, not a rewrite:
- **Per-user usage record** already exists (owner portal usage dashboard) — reuse it as the metering source of truth.
- Enforce soft caps at the trust boundary: note count and AI token/day per user (the same caps that protect
  against AI-cost blowout in §5.2 double as the free-tier ceiling).
- Represent limits as named config (`FREE_TIER_*`), not hardcoded, so a future `PRO` tier only adds a plan lookup.
- No Stripe, no plan UI, no gating logic in v1 — only the enforced ceiling and the config seam.

## 6. Staged roadmap (owner-defined order)

### Stage 1 — Product plan (this document)
Vision, positioning, architecture direction, scope boundaries. **Deliverable: this PRD.** Approval gate
before any build.

### Stage 2 — Production multi-user hardening + AI foundation
Ship the substance first — make the product actually *do the new thing* and survive real signups.
- Harden public-instance mode for real signups: signup UX (fix the misleading error already found),
  email verification/password reset, abuse/rate limiting, backups.
- Enforce free-tier caps (§5.4) — note count + AI token/day per user, as named config.
- Stand up the AI layer foundation: embedding store, indexing worker, provider interface, semantic search.
- Gate: a stranger can sign up and use semantic search on their own private notes safely, within enforced caps.

### Stage 3 — Rebrand + UX
Make it look and feel like its own product, not a memos skin — once it already does something memos can't.
- Name (final, with a clear domain + trademark check), logo, color/type system, landing/marketing surface, app theming.
- Visual direction: **dark luxury** (near-black layered surfaces, single vivid accent, subtle depth/glow,
  monospace numerics — Vercel/Raycast register).
- Opinionated redesign of the core note surfaces (per web design-quality standards).
- No backend/architecture changes in this stage.
- Gate: a believable product screenshot a stranger would not recognize as memos.

### Later (not in first three stages)
- RAG chat, AI organize, AI writing assist (iterative on the Stage 3 foundation).
- Billing / paid tiers (Stripe), plan gating.
- Collaboration / workspaces (only if teams become a goal — triggers the deferred org model).

## 7. Non-goals for the first three stages
- Organizations, workspaces, teams, shared spaces, seat management.
- Billing, subscriptions, Stripe, plan UI, invoices. (Free-tier *caps* are in scope per §5.4; charging for them is not.)
- Mobile native apps (web-responsive only).
- Replacing memos' auth, storage, or API architecture.
- Real-time collaborative editing.

## 8. Key risks
- **Upstream drift:** heavy divergence makes pulling upstream fixes painful. Mitigate by extending, not
  rewriting, and isolating new code in new packages/modules.
- **AI cost:** uncontrolled usage on a free tier. Mitigate with hard per-user caps from day one.
- **Data isolation:** an AI leak across users is a product-ending trust failure. Treat user-scoping of
  all AI retrieval as a CRITICAL, tested boundary.
- **Scope creep into orgs/billing too early:** resist until the personal AI product is actually good.

## 9. Open decisions (needed before Stage 3, not before Stage 2)
- Production database: commit to Postgres for the hosted product (required for pgvector)?
- LLM/embedding provider and model choice + budget ceiling.
- Hosting target (VPS / container platform) and backup strategy.
