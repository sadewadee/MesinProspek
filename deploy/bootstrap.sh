#!/usr/bin/env bash
# Bootstrap the single MesinProspek server (docs/06-ci-cd.md §9, WP-05). Idempotent: safe to re-run,
# and the first step of a rebuild (docs/09-runbook-operasional.md RB-11).
#
# Run from a clone of the repo on Ubuntu LTS, as root the first time, with sudo afterwards
# (this script disables root login):
#   scp -r deploy root@<ip>:  &&  ssh root@<ip> bash deploy/bootstrap.sh
# Inputs that never live in the repo (fill from the founder's password manager, 06 §7):
#   /opt/ops/.env           keys and format in deploy/ops/.env.example
#   /opt/ops/rclone.conf    rclone remote `offsite` (06 §8), root, mode 600
# Input in the repo: deploy/admin-keys/<user>.pub, the SSH public key of each admin (one user per admin).
set -euo pipefail

# Pinned versions (06: every image and tool is pinned). Images are pinned in the compose files.
DOCKER_VERSION=29.8.2      # docker-ce, docker-ce-cli
COMPOSE_VERSION=5.6.0      # docker-compose-plugin
RUNNER_VERSION=19.4.1      # gitlab-runner host package (runner `deploy`); same as the image in ops/compose.yaml

SRC=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
OPS=/opt/ops

log() { printf '\n==> %s\n' "$*"; }
die() { printf '\nSTOP: %s\n' "$*" >&2; exit 1; }
# env_get KEY FILE: value of KEY in a dotenv file, without surrounding double quotes
env_get() { local v; v=$(sed -n "s/^$1=//p" "$2" | tail -n 1); v=${v%\"}; printf '%s' "${v#\"}"; }
# put SRC DST MODE [OWNER]: install a file only when its content differs; sets changed=1 when it did
put() {
  if ! cmp -s "$1" "$2"; then install -m "$3" -o "${4:-root}" -g "${4:-root}" "$1" "$2"; changed=1; fi
}

# --- 1. Preconditions: nothing on the system changes before these pass ---------------------------
[ "$(id -u)" -eq 0 ] || die "run as root (first run) or with sudo"
# shellcheck source=/dev/null
. /etc/os-release
[ "${ID:-}" = ubuntu ] || die "Ubuntu LTS expected, found ${PRETTY_NAME:-unknown}"
[ -f "$SRC/ops/compose.yaml" ] || die "run this script from a clone of the repo (deploy/ops/ not found)"

missing=()
[ -f "$OPS/.env" ] || missing+=("$OPS/.env: cp $SRC/ops/.env.example $OPS/.env, then replace every CHANGE_ME")
[ -f "$OPS/rclone.conf" ] || missing+=("$OPS/rclone.conf: remote [offsite] (sftp with known_hosts_file, or ftp with explicit_tls = true)")
compgen -G "$SRC/admin-keys/*.pub" >/dev/null || missing+=("$SRC/admin-keys/<user>.pub: SSH public key of each admin")
if [ ${#missing[@]} -gt 0 ]; then
  install -d -m 755 "$OPS"
  printf '  - %s\n' "${missing[@]}" >&2
  die "fill in the inputs above from the founder's password manager, then re-run (docs/06-ci-cd.md §7, §9)"
fi

bad=()
while read -r key; do
  val=$(env_get "$key" "$OPS/.env")
  if [ -z "$val" ] || [[ $val == *CHANGE_ME* ]]; then bad+=("$key"); fi
done < <(sed -n 's/^\([A-Z0-9_]*\)=.*/\1/p' "$SRC/ops/.env.example")
[ ${#bad[@]} -eq 0 ] || die "set these keys in $OPS/.env (see $SRC/ops/.env.example): ${bad[*]}"
ADMIN_IPS=$(env_get MSP_ADMIN_IPS "$OPS/.env")

admins=()
for f in "$SRC"/admin-keys/*.pub; do
  u=$(basename "$f" .pub)
  [[ $u =~ ^[a-z][a-z0-9-]{0,30}$ ]] || die "admin key file name must be <user>.pub with a valid user name: $f"
  admins+=("$u")
done

# --- 2. Packages -------------------------------------------------------------------------------------
log "Packages: Docker $DOCKER_VERSION + compose $COMPOSE_VERSION, gitlab-runner $RUNNER_VERSION, rclone, ufw"
export DEBIAN_FRONTEND=noninteractive
apt-get update -q
apt-get install -y -q ca-certificates curl git iproute2 python3 rclone ufw unattended-upgrades
arch=$(dpkg --print-architecture)
install -d -m 755 /etc/apt/keyrings
[ -s /etc/apt/keyrings/docker.asc ] || curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
[ -s /etc/apt/keyrings/gitlab-runner.asc ] || curl -fsSL https://packages.gitlab.com/runner/gitlab-runner/gpgkey -o /etc/apt/keyrings/gitlab-runner.asc
echo "deb [arch=$arch signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $VERSION_CODENAME stable" \
  > /etc/apt/sources.list.d/docker.list
echo "deb [arch=$arch signed-by=/etc/apt/keyrings/gitlab-runner.asc] https://packages.gitlab.com/runner/gitlab-runner/ubuntu/ $VERSION_CODENAME main" \
  > /etc/apt/sources.list.d/gitlab-runner.list
apt-get update -q
suffix="-1~ubuntu.$VERSION_ID~$VERSION_CODENAME"
apt-get install -y -q --allow-change-held-packages --allow-downgrades \
  "docker-ce=5:$DOCKER_VERSION$suffix" "docker-ce-cli=5:$DOCKER_VERSION$suffix" containerd.io docker-buildx-plugin \
  "docker-compose-plugin=$COMPOSE_VERSION$suffix" "gitlab-runner=$RUNNER_VERSION-1"
apt-mark hold docker-ce docker-ce-cli docker-compose-plugin gitlab-runner gitlab-runner-helper-images >/dev/null
# Security updates only (Ubuntu default origins); reboots stay manual (09 §6)
printf 'APT::Periodic::Update-Package-Lists "1";\nAPT::Periodic::Unattended-Upgrade "1";\n' > /etc/apt/apt.conf.d/20auto-upgrades

log "Docker: json-file log rotation 50m x 3 (06 §9)"
daemon_json='{ "log-driver": "json-file", "log-opts": { "max-size": "50m", "max-file": "3" } }'
if [ "$(cat /etc/docker/daemon.json 2>/dev/null)" != "$daemon_json" ]; then
  printf '%s\n' "$daemon_json" > /etc/docker/daemon.json
  systemctl restart docker      # existing containers keep their old log settings until recreated
fi
systemctl enable --now docker >/dev/null

# --- 3. Users ------------------------------------------------------------------------------------------
log "Users: deploy (runner deploy) and admins ${admins[*]} (06 §9)"
id deploy >/dev/null 2>&1 || useradd --create-home --shell /bin/bash deploy
usermod -aG docker deploy
rm -f /home/deploy/.bash_logout     # its clear_console breaks the runner's shell executor
for u in "${admins[@]}"; do
  id "$u" >/dev/null 2>&1 || useradd --create-home --shell /bin/bash "$u"
  usermod -aG sudo,docker,deploy "$u"
  install -d -m 700 -o "$u" -g "$u" "/home/$u/.ssh"
  install -m 600 -o "$u" -g "$u" "$SRC/admin-keys/$u.pub" "/home/$u/.ssh/authorized_keys"
done
# Admins log in with keys only and have no password, so sudo must not ask for one
echo '%sudo ALL=(ALL:ALL) NOPASSWD: ALL' > /etc/sudoers.d/90-msp-admins
chmod 440 /etc/sudoers.d/90-msp-admins
visudo -cqf /etc/sudoers.d/90-msp-admins
for u in $(getent group sudo | cut -d: -f4 | tr ',' ' '); do
  [ -f "$SRC/admin-keys/$u.pub" ] || echo "WARNING: $u is in group sudo without admin-keys/$u.pub (person left? docs/09 §2)"
done

# --- 4. Directories (README § Kosakata teknis bersama) --------------------------------------------------
log "Directories: /opt/msp, /opt/msp-staging, $OPS"
install -d -m 755 -o deploy -g deploy /opt/msp /opt/msp-staging "$OPS"
install -d -m 750 -o 999 -g 999 /opt/msp/pgbackrest /opt/msp-staging/pgbackrest   # 999 = postgres in the image (06 §8)
install -d -m 755 "$OPS/textfile"                                                 # offsite.sh -> node-exporter
install -d -m 2775 -o deploy -g deploy "$OPS/gmaps-dump"                          # weekly gmaps drops (09 RB-15)
mkdir -p "$OPS/gitlab/config" "$OPS/gitlab/logs" "$OPS/gitlab/data" "$OPS/gitlab-backups" "$OPS/gitlab-runner"

# --- 5. Ops stack files, cron, secrets' permissions -----------------------------------------------------
log "Ops stack files -> $OPS, cron /etc/cron.d/msp-offsite"
changed=0
for f in Caddyfile prometheus.yml alerts.yml alertmanager.yml alertmanager-start.sh grafana-datasource.yml; do
  put "$SRC/ops/$f" "$OPS/$f" 644 deploy
done
config_changed=$changed           # these files are bind-mounted: running services need a restart
put "$SRC/ops/compose.yaml" "$OPS/compose.yaml" 644 deploy
put "$SRC/ops/.env.example" "$OPS/.env.example" 644 deploy
put "$SRC/ops/runner-ci.template.toml" "$OPS/gitlab-runner/ci.template.toml" 644
put "$SRC/ops/offsite.sh" "$OPS/offsite.sh" 755
put "$SRC/ops/msp-offsite.cron" /etc/cron.d/msp-offsite 644
chown deploy:deploy "$OPS/.env" && chmod 600 "$OPS/.env"            # read by docker compose in /opt/ops (also backup.sh)
chown root:root "$OPS/rclone.conf" && chmod 600 "$OPS/rclone.conf"  # read only by root cron (06 §7)

# --- 6. Runners: concurrent = 1 (06 §2); registration is manual (06 §9) -------------------------------
log "GitLab runners: ci (container) and deploy (host, shell, user deploy)"
set_concurrent() {
  if [ ! -s "$1" ]; then printf 'concurrent = 1\ncheck_interval = 0\n' > "$1"; chmod 600 "$1"
  elif grep -q '^concurrent' "$1"; then sed -i 's/^concurrent *=.*/concurrent = 1/' "$1"
  else sed -i '1i concurrent = 1' "$1"; fi
}
set_concurrent "$OPS/gitlab-runner/config.toml"
install -d -m 700 /etc/gitlab-runner
set_concurrent /etc/gitlab-runner/config.toml
# The package installs its service as user gitlab-runner (again on every package install): switch to deploy
if ! systemctl cat gitlab-runner 2>/dev/null | grep -Eq -- '--user"? "?deploy"?( |$)'; then
  gitlab-runner uninstall >/dev/null 2>&1 || true
  gitlab-runner install --user deploy --working-directory /home/deploy --config /etc/gitlab-runner/config.toml
  systemctl daemon-reload
  systemctl restart gitlab-runner
fi
systemctl enable --now gitlab-runner >/dev/null

# --- 7. SSH: keys only, no root login (06 §9) ----------------------------------------------------------
log "SSH: PasswordAuthentication no, PermitRootLogin no"
# 10- sorts before cloud-init's 50- drop-in; sshd keeps the first value it reads
printf 'PasswordAuthentication no\nKbdInteractiveAuthentication no\nPermitRootLogin no\n' > /etc/ssh/sshd_config.d/10-msp.conf
mkdir -p /run/sshd                  # missing until ssh.service first runs (socket activation); sshd -t needs it
sshd -t
systemctl try-reload-or-restart ssh.service

# --- 8. Firewall: 80/443 open, 22 only from MSP_ADMIN_IPS (06 §9, 09 §2) -------------------------------
log "Firewall (ufw): 80/443, SSH from: $ADMIN_IPS"
# Valid IPs/CIDRs, and refuse to lock out: at least one open SSH session must come from the allowlist
python3 - "$ADMIN_IPS" "$(ss -Htn state established '( sport = :22 )' | awk '{print $4}')" <<'PY' || die "fix MSP_ADMIN_IPS in /opt/ops/.env"
import ipaddress, sys
try:
    nets = [ipaddress.ip_network(n, strict=False) for n in sys.argv[1].split()]
except ValueError as e:
    sys.exit(f"MSP_ADMIN_IPS: {e}")
peers = []
for p in sys.argv[2].split():
    ip = ipaddress.ip_address(p.rsplit(":", 1)[0].strip("[]"))
    if ip.version == 6 and ip.ipv4_mapped:
        ip = ip.ipv4_mapped
    peers.append(ip)
if peers and not any(ip in n for ip in peers for n in nets if ip.version == n.version):
    sys.exit(f"no current SSH session comes from MSP_ADMIN_IPS {sys.argv[1]} (sessions: {', '.join(map(str, peers))}); the firewall would lock you out")
PY
ufw default deny incoming >/dev/null
ufw default allow outgoing >/dev/null
# Remove SSH rules that are not an allowlisted source (open 22, or an IP removed from the list)
{ ufw show added | grep -E '^ufw (allow|limit) (22(/tcp)?|OpenSSH|.* port 22( .*)?)$' || true; } | while read -r rule; do
  ip=$(sed -n 's/^ufw allow from \([^ ]*\) to any port 22 proto tcp$/\1/p' <<<"$rule")
  if [ -z "$ip" ] || [[ " $ADMIN_IPS " != *" $ip "* ]]; then
    # shellcheck disable=SC2086 # the rule is a list of ufw words
    ufw delete ${rule#ufw } >/dev/null
  fi
done
for ip in $ADMIN_IPS; do ufw allow from "$ip" to any port 22 proto tcp >/dev/null; done
ufw allow 80/tcp >/dev/null
ufw allow 443/tcp >/dev/null
ufw --force enable >/dev/null

# --- 9. Ops stack up (creates networks edge-production, edge-staging, gmaps) ---------------------------
log "Ops stack: docker compose up -d in $OPS"
cd "$OPS"
docker compose up -d --remove-orphans
if [ "$config_changed" -eq 1 ]; then docker compose restart caddy prometheus alertmanager grafana; fi

log "Done. One-time manual steps: deploy/README.md (docs/06-ci-cd.md §9)"
