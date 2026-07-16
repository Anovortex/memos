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
