# Deployment

Push to `main` → GitHub Actions builds one Docker image (api + worker), pushes it to
GHCR, then SSHes into the server and restarts the stack. DB schema (GORM `AutoMigrate`
on api boot) and Kafka topics (`kafka-init`) update themselves — no manual migration.

Stack on the server (`docker-compose.prod.yml`): `postgres`, `kafka`, `kafka-init`,
`backend` (api), `worker`. Postgres + Kafka data persist in named volumes.

## One-time server setup

1. **Install Docker + compose plugin** (Ubuntu):
   ```bash
   curl -fsSL https://get.docker.com | sh
   sudo usermod -aG docker $USER   # re-login after this
   ```

2. **CI SSH key** — on your machine:
   ```bash
   ssh-keygen -t ed25519 -f swapngo_deploy -N ""
   ssh-copy-id -i swapngo_deploy.pub <user>@<server>   # or append .pub to ~/.ssh/authorized_keys
   ```
   Add the **private** key (`swapngo_deploy`) to the repo secret `SERVER_SSH_KEY`.

3. **Deploy dir + files** on the server:
   ```bash
   mkdir -p ~/swapngo && cd ~/swapngo
   # copy docker-compose.prod.yml here (scp, git clone, or paste)
   cp .env.production.example .env   # then edit .env with REAL secrets
   ```

4. **First run**:
   ```bash
   cd ~/swapngo
   docker login ghcr.io        # only if the GHCR package is private
   docker compose -f docker-compose.prod.yml up -d
   ```
   After this, every push to `main` redeploys automatically.

## Required GitHub repo secrets

Settings → Secrets and variables → Actions:

| Secret           | Value                                     |
| ---------------- | ----------------------------------------- |
| `SERVER_HOST`    | server IP / hostname                      |
| `SERVER_USER`    | SSH user (the deploy user)                |
| `SERVER_SSH_KEY` | private SSH key (matches authorized_keys) |
| `SERVER_PORT`    | SSH port (optional, defaults to 22)       |

GHCR auth uses the built-in `GITHUB_TOKEN` — no extra secret needed. If the published
package is **private**, either make it public (repo → Packages → package settings) or
run `docker login ghcr.io` once on the server with a read-scoped PAT.

## Operations

```bash
cd ~/swapngo
docker compose -f docker-compose.prod.yml ps                 # status
docker compose -f docker-compose.prod.yml logs -f backend    # api logs (look for "Database migration completed successfully!")
docker compose -f docker-compose.prod.yml logs kafka-init    # lists created topics
docker compose -f docker-compose.prod.yml down               # stop (volumes kept)
```
