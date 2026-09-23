#!/usr/bin/env bash
#
# Backup for the Memos production Compose deployment (rootless Podman or Docker).
#
#   1. pg_dump -Fc run inside the postgres container (no client tools on the host)
#   2. tar.gz of the Memos data directory or volume (local attachments)
#   3. sha256 sidecars for both files
#   4. prune backups older than KEEP_DAYS (default 14)
#
# Exits non-zero if any step fails so the systemd timer records the failure.
# Install the timer with deploy/memos-backup.{service,timer}; see README.
#
# Environment (all optional):
#   COMPOSE_DIR   directory holding compose.yaml and .env   (default: this script's dir)
#   COMPOSE       compose command                           (default: "docker compose")
#   BACKUP_DIR    where backups are written                 (default: $HOME/backups/memos)
#   KEEP_DAYS     retention in days                         (default: 14)
#
# Copy BACKUP_DIR off the host afterwards (rsync over SSH, rclone); a backup that
# only lives on the machine it protects is not a backup.

set -uo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
COMPOSE="${COMPOSE:-docker compose}"
BACKUP_DIR="${BACKUP_DIR:-$HOME/backups/memos}"
KEEP_DAYS="${KEEP_DAYS:-14}"

log() { echo "[$(date -Is)] $*"; }
fail() { log "FAIL: $*"; FAILED=1; }
FAILED=0

cd "$COMPOSE_DIR" || { echo "cannot cd to $COMPOSE_DIR"; exit 1; }
[[ -f .env ]] && set -a && . ./.env && set +a
PGUSER="${POSTGRES_USER:-memos}"
PGDB="${POSTGRES_DB:-memos}"
DATA_SOURCE="${MEMOS_DATA_SOURCE:-memos_data}"
PROJECT="${COMPOSE_PROJECT_NAME:-memos-production}"

mkdir -p "$BACKUP_DIR"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
DUMP="$BACKUP_DIR/memos-$STAMP.dump"
DATA="$BACKUP_DIR/memos-data-$STAMP.tar.gz"

# 1. Database dump from inside the container (unix-socket trust auth, no password needed).
log "dumping $PGDB -> $DUMP"
if $COMPOSE exec -T postgres pg_dump -Fc -U "$PGUSER" "$PGDB" > "$DUMP.tmp"; then
  size=$(stat -c%s "$DUMP.tmp" 2>/dev/null || stat -f%z "$DUMP.tmp")
  if [[ "$size" -lt 1024 ]]; then
    fail "dump is only ${size} bytes"; rm -f "$DUMP.tmp"
  else
    mv "$DUMP.tmp" "$DUMP"
  fi
else
  fail "pg_dump exited non-zero"; rm -f "$DUMP.tmp"
fi

# 2. Data directory (attachments). A bind-mounted path is tarred directly, via
#    `podman unshare` when rootless so container-owned files are readable; a
#    named volume is tarred through a throwaway container.
log "archiving data ($DATA_SOURCE) -> $DATA"
if [[ "$DATA_SOURCE" == /* ]]; then
  if command -v podman >/dev/null 2>&1 && podman info --format '{{.Host.Security.Rootless}}' 2>/dev/null | grep -q true; then
    podman unshare tar -czf "$DATA.tmp" -C "$DATA_SOURCE" . || fail "tar of $DATA_SOURCE exited non-zero"
  else
    tar -czf "$DATA.tmp" -C "$DATA_SOURCE" . || fail "tar of $DATA_SOURCE exited non-zero"
  fi
else
  engine="${COMPOSE%% *}"
  $engine run --rm -v "${PROJECT}_${DATA_SOURCE}:/data:ro" alpine:3.21 tar -czf - -C /data . > "$DATA.tmp" \
    || fail "tar of volume ${PROJECT}_${DATA_SOURCE} exited non-zero"
fi
if [[ -s "$DATA.tmp" ]]; then mv "$DATA.tmp" "$DATA"; else rm -f "$DATA.tmp"; fail "data archive is empty"; fi

# 3. Checksums.
for f in "$DUMP" "$DATA"; do
  [[ -f "$f" ]] || continue
  ( cd "$BACKUP_DIR" && sha256sum "$(basename "$f")" > "$(basename "$f").sha256" ) || fail "sha256 for $f"
done

# 4. Prune.
find "$BACKUP_DIR" -maxdepth 1 -type f \( -name 'memos-*.dump*' -o -name 'memos-data-*.tar.gz*' \) -mtime +"$KEEP_DAYS" -print -delete \
  | sed 's/^/pruned: /'

if [[ "$FAILED" -ne 0 ]]; then log "backup FAILED"; exit 1; fi
log "backup OK: $(du -sh "$DUMP" "$DATA" 2>/dev/null | tr '\n' ' ')"
