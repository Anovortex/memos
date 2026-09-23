# Fork schema migrations and the 0.31.2 upgrade path

## Where fork migrations live now

Upstream reset its migration baseline at schema **0.31.8** (shipped by upstream
v0.31.0) and dropped every historical SemVer script from the binary. New
migrations use the calendar scheme `store/migration/<driver>/YY.MM/NN__name.sql`
(see `store/migration/README.md`). The fork's two tables therefore live at:

| Driver   | Files                                                                            | Records        |
|----------|----------------------------------------------------------------------------------|----------------|
| sqlite   | `store/migration/sqlite/26.09/00__memo_embedding.sql`, `01__user_ai_usage.sql`   | 26.9.1, 26.9.2 |
| mysql    | `store/migration/mysql/26.09/00__memo_embedding.sql`, `01__user_ai_usage.sql`    | 26.9.1, 26.9.2 |
| postgres | `store/migration/postgres/26.09/00__memo_embedding.sql`, `01__user_ai_usage.sql` | 26.9.1, 26.9.2 |

Both files are idempotent (`CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT
EXISTS`; MySQL declares the index inline). They no longer declare a foreign key
to `memo(id)`: SQLite runs with `foreign_keys(0)` anyway, and the store deletes
the embedding row explicitly in `DeleteMemoWithPolicy`. Dropping the FK keeps
future upstream `memo` table rebuilds (SQLite does `DROP TABLE memo`) safe.
`LATEST.sql` for all three drivers carries the same definitions, so a fresh
install lands directly on **26.9.2**. `TestFreshInstall` and the calendar guard
case in `store/test` assert this.

## A live database that records 0.31.2

The production database was migrated by the pre-merge fork binary, which
numbered these tables `0.31/00` and `0.31/01`, so `system_setting.BASIC`
records `schemaVersion = "0.31.2"`. Its actual schema is upstream **0.30.1**
(the last upstream script at the fork point, `0.30/00__user_tag_setting.sql`)
plus `memo_embedding` and `user_ai_usage`. It has none of upstream's real
0.31 series (`0.31/00` through `0.31/07`: memo views rename, storage-setting
expansion, reaction.memo_id, Spaces, space member status, space payload,
unique email).

What each binary does with it:

* **This merged binary**: `checkMinimumUpgradeVersion` compares `0.31.2` with
  the baseline `0.31.8` and **refuses to start**:
  `database schema "0.31.2" is too old to upgrade directly; minimum supported
  schema is 0.31.8. First upgrade to v0.31.0 ...`. No data is touched.
* **Upstream v0.31.0 as-is**: it would accept the database but apply only
  `0.31/02` through `0.31/07` (versions greater than `0.31.2`), silently
  skipping `0.31/00` (SHORTCUTS -> MEMO_VIEWS user-setting rewrite) and
  `0.31/01` (storage-setting expansion). That leaves memo views and storage
  configuration broken. Do not run it against the version marker as recorded.

### Recommended path (Postgres production; also valid for SQLite/MySQL)

1. Stop the fork server and take a backup (`deploy/backup.sh`).
2. Reset the recorded version to the schema the database really has:

   ```sql
   -- Postgres; schemaVersion is a JSON key inside the BASIC setting value.
   UPDATE system_setting
   SET value = jsonb_set(value::jsonb, '{schemaVersion}', '"0.30.1"')::text
   WHERE name = 'BASIC';
   ```

   SQLite: `UPDATE system_setting SET value = json_set(value, '$.schemaVersion', '0.30.1') WHERE name = 'BASIC';`
3. Start upstream **v0.31.0** once (image `neosmemo/memos:0.31.0`) against the
   database and let it finish. It applies `0.31/00` .. `0.31/07` and records
   `0.31.8`. The extra fork tables do not interfere: the Postgres scripts use
   `ALTER TABLE` and never drop `memo` or `user`. On SQLite, `0.31/03` and
   `0.31/07` rebuild `memo`/`user`; with `foreign_keys(0)` the old FK on
   `memo_embedding` is inert, and embedding rows are simply re-indexed later.
   v0.31.0 also performs the Unicode email canonicalization that `0.31/07`
   relies on, which is why this step must use that release rather than a
   hand-run SQL script.
4. Verify sign-in, memo views, and storage settings on v0.31.0.
5. Start the merged fork binary. `preMigrate` accepts `0.31.8`, applies
   `26.09/00` and `26.09/01` (both no-ops on this database thanks to
   `IF NOT EXISTS`), records `26.9.2`, and the embedding indexer resumes.

Optional: `ALTER TABLE memo_embedding DROP CONSTRAINT memo_embedding_memo_id_fkey;`
on Postgres brings the live table in line with the FK-free definition. It is
not required for the upgrade.

### Why not an automated bridge

A bridge would have to embed upstream's 0.31 scripts plus the Go-side email
canonicalization from v0.31.0. Running the real v0.31.0 release once is the
tested path; a bridge can be added later if more fork databases at 0.31.x appear.
