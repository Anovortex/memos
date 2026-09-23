# Production Compose deployment

This deployment runs Memos with PostgreSQL and server-controlled persistent
volumes. Memos binds to server loopback only; expose it through the approved
reverse proxy or Cloudflare Tunnel.

## First deployment

```bash
cp .env.example .env
chmod 600 .env
# Set MEMOS_IMAGE in .env to an immutable image tag or digest.
install -d -m 700 secrets
password="$(openssl rand -hex 32)"
printf '%s' "$password" > secrets/postgres_password
printf 'postgres://memos:%s@postgres:5432/memos?sslmode=disable' "$password" > secrets/memos_dsn
chmod 600 secrets/*
unset password

docker compose config
docker compose pull memos
docker compose pull postgres
docker compose up -d
docker compose ps
```

## Cloudflare Tunnel

Create a dedicated tunnel and DNS route, then copy
`cloudflared.example.yml` to the ignored `cloudflared.yml` and fill in the
tunnel ID and hostname. Store its credential JSON at
`secrets/cloudflared_credentials.json`.

Start the public tunnel with:

```bash
docker compose -f compose.yaml -f compose.cloudflare.yaml config
docker compose -f compose.yaml -f compose.cloudflare.yaml pull
docker compose -f compose.yaml -f compose.cloudflare.yaml up -d
docker compose -f compose.yaml -f compose.cloudflare.yaml ps
```

The Memos port remains bound to server loopback; only the Cloudflare Tunnel
publishes it.

After changing `cloudflared.yml` or its credential JSON, force the initializer
and tunnel to consume the new files:

```bash
docker compose -f compose.yaml -f compose.cloudflare.yaml up -d \
  --force-recreate cloudflared-secret-init cloudflared
```

When permanently retiring the tunnel, remove the
`memos-production_cloudflared_config` volume after stopping the Compose project
so the copied credential does not remain at rest.

The persistent volumes are:

- `memos-production_postgres_data` for PostgreSQL.
- `memos-production_memos_data` for local attachments and Memos runtime data.

Do not commit `.env` or `secrets/`.

Back up PostgreSQL with `pg_dump -Fc`, and verify that the dump restores into a
separate database. Do not copy its live data volume as a backup. Back up
`memos-production_memos_data` separately because a database dump does not
contain locally stored attachments. A coordinated stopped-volume snapshot or
PostgreSQL PITR setup is also valid when accompanied by a tested restore plan.

## Backups

`backup.sh` writes a `pg_dump -Fc` of the database and a tar.gz of the Memos
data directory (attachments), each with a `.sha256` sidecar, into
`~/backups/memos`, and prunes files older than 14 days. It runs `pg_dump`
inside the postgres container, so the host needs no PostgreSQL client. On a
rootless Podman host it reads the bind-mounted data directory through
`podman unshare`.

Install the nightly timer as the deploy user (units assume the checkout at
`~/repos/memos`; edit the paths in `memos-backup.service` otherwise):

```bash
install -Dm644 deploy/memos-backup.service deploy/memos-backup.timer -t ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now memos-backup.timer
loginctl enable-linger "$USER"      # so the timer runs without a login session
```

Run it once by hand and verify the dump is restorable:

```bash
systemctl --user start memos-backup.service
journalctl --user -u memos-backup.service -n 20
pg_restore --list ~/backups/memos/memos-*.dump | head
```

Copy `~/backups/memos` off the host on a schedule (rsync over SSH or rclone).
A backup that only lives on the machine it protects is not a backup.
