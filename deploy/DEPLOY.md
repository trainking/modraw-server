# Deployment Guide — modraw-server

## Directory Structure

```
deploy/
  build.sh                  Local build script (cross-compile Go for Linux)
  bin/                      Output: compiled binary (gitignored)
  ansible/
    ansible.cfg             Ansible config
    inventory/hosts.yml     Server inventory
    group_vars/all.yml      Non-sensitive variables (paths, ports)
    .env.example            Template for secrets file
    .env                    Local secrets file (gitignored, never committed)
    load_secrets.yml        Task to load .env into Ansible vars
    playbook.yml            Main playbook
    roles/
      postgresql/           PostgreSQL installation + database setup
      modraw-server/        App deployment (binary, systemd, Nginx)
```

## Quick Deploy

### 1. Create secrets file

```bash
cd deploy/ansible
cp .env.example .env
```

Edit `.env` with your real values:

```ini
PG_PASSWORD=your-strong-db-password-here
JWT_SECRET=your-random-secret-here
```

The `.env` file is **gitignored** — it will never be committed.

### 2. Edit non-sensitive variables

Update `deploy/ansible/group_vars/all.yml`:

```yaml
server_domain: "gmc.example.com"   # your actual domain
# All other values have sensible defaults
```

### 3. Build the binary

From the project root:

```bash
bash deploy/build.sh
```

This cross-compiles `modraw-server` for `linux/amd64` and places it at `deploy/bin/modraw-server`.

### 4. Run the playbook

```bash
cd deploy/ansible

# First-time: PostgreSQL + app
ansible-playbook playbook.yml --tags setup,deploy

# Subsequent deploys: app only
ansible-playbook playbook.yml --tags deploy
```

## How Secrets Work

1. `playbook.yml` calls `load_secrets.yml` as the first step (with `tags: [always]`)
2. `load_secrets.yml` parses the local `.env` file and stores values as the `deploy` fact
3. Templates and tasks reference secrets via `{{ deploy.PG_PASSWORD }}`, `{{ deploy.JWT_SECRET }}`
4. The `.env` file is in `.gitignore` — it never touches the repository

Add new secrets by adding a key to `.env`, then referencing `{{ deploy.YOUR_KEY }}` in templates.

## Tag Summary

| Tag | What it does |
|---|---|
| `setup` | Install PostgreSQL, create user + database |
| `deploy` | Upload binary + migrations, deploy env/systemd/Nginx |
| `postgres` | PostgreSQL only (alias for setup) |
| `app` | App only (alias for deploy) |

## What Gets Deployed

| Component | Path / Details |
|---|---|
| Binary | `/opt/modraw-server/modraw-server` |
| Migrations | `/opt/modraw-server/migrations/` |
| Env file | `/opt/modraw-server/.env` (mode 0640) |
| systemd unit | `/etc/systemd/system/modraw-server.service` |
| Nginx site | `/etc/nginx/sites-enabled/modraw-server` |
| App user | `modraw:modraw` (system user, no login) |

## Post-Deploy Verification

```bash
# Check service status
systemctl status modraw-server

# Check logs
journalctl -u modraw-server -f

# Test health endpoint
curl http://localhost:8080/health

# Test through Nginx
curl http://gmc.example.com/health

# Check WebSocket upgrade (should return 400 — means it reaches the server)
curl -i "http://localhost:8080/ws"
```

## Updating the Deployment

1. Make code changes
2. Rebuild: `bash deploy/build.sh`
3. Redeploy: `ansible-playbook playbook.yml --tags deploy`

The handler will restart the service automatically.

## Troubleshooting

**Service fails to start:**
```bash
journalctl -u modraw-server -n 50 --no-pager
```

**Database connection refused:**
```bash
# Check PostgreSQL is running
systemctl status postgresql
# Test connection
sudo -u postgres psql -c "\l"
sudo -u postgres psql -c "SELECT 1" -U modraw -d modraw
```

**Nginx 502 Bad Gateway:**
```bash
# Check app is listening
ss -tlnp | grep 8080
# Check Nginx error log
tail -f /var/log/nginx/error.log
```

**Permission denied on upload:**
Make sure the SSH user has `become: yes` (sudo) in the inventory, or set passwordless sudo on the server.
