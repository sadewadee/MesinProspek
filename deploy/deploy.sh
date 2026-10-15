#!/usr/bin/env bash
# Deploy one app environment on this server (docs/06-ci-cd.md §4). Run by runner `deploy` (shell executor,
# user deploy) from a checkout of the commit being deployed, in the jobs deploy-staging and
# deploy-production of .gitlab-ci.yml. A rollback is the same script with the previous release's tag:
# run that pipeline's deploy job again (docs/09 RB-01, RB-02).
#
#   deploy/deploy.sh <staging|production> <image-tag>
#
# 1. Writes <dir>/.env (mode 600) from the CI/CD variables named in deploy/.env.example (06 §7).
# 2. Copies the compose files, postgres/ and kratos/ to <dir> (/opt/msp-staging or /opt/msp).
# 3. Pulls the images and runs `msp migrate` and Kratos' migrations. If they fail, the running version stays.
# 4. `docker compose up -d --wait`, then checks $MSP_PUBLIC_URL/readyz through Caddy.
# It never prints a value from .env.
set -euo pipefail

usage() { echo "usage: deploy/deploy.sh <staging|production> <image-tag>" >&2; exit 2; }
die() { printf 'deploy: %s\n' "$*" >&2; exit 1; }
log() { printf '\n==> %s\n' "$*"; }

[ $# -eq 2 ] || usage
env=$1 tag=$2
case $env in
  production) dir=/opt/msp ;;
  staging) dir=/opt/msp-staging ;;
  *) usage ;;
esac
[[ $tag =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$ ]] || die "not a valid image tag: $tag"
src=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
[ "$(id -un)" = deploy ] || die "run as user deploy (runner deploy), so $dir stays owned by deploy (06 §9)"
[[ -d $dir && -w $dir ]] || die "$dir must exist and be writable by deploy: run deploy/bootstrap.sh (06 §9)"
: "${CI_REGISTRY:?}" "${CI_REGISTRY_IMAGE:?}"
: "${CI_DEPLOY_USER:?create the deploy token gitlab-deploy-token (read_registry) in GitLab, deploy/README.md}"
: "${CI_DEPLOY_PASSWORD:?create the deploy token gitlab-deploy-token (read_registry) in GitLab, deploy/README.md}"
umask 022
trap 'echo "deploy: FAILED. Containers that were not recreated keep running the previous version; to go back, run the deploy job of the previous release again (docs/09 RB-02)." >&2' ERR

log "Writing $dir/.env for $env, image tag $tag (keys: deploy/.env.example)"
declare -A own=([MSP_ENV]=$env [MSP_IMAGE_REPO]=$CI_REGISTRY_IMAGE [MSP_IMAGE_TAG]=$tag [COMPOSE_FILE]=compose.yaml)
[ "$env" = production ] || own[COMPOSE_FILE]=compose.yaml:compose.staging.yaml
lines=() missing=()
while IFS='=' read -r key example; do
  if [[ -v own[$key] ]]; then val=${own[$key]}
  elif [ -n "${!key:-}" ]; then val=${!key}        # CI/CD variable of the job's environment scope
  elif [[ $example == *CHANGE_ME* ]]; then missing+=("$key"); continue
  elif [ -z "$example" ]; then continue
  else val=$example
  fi
  [[ $val != *[\'$'\n']* ]] || die "the value of $key contains a single quote or a line break"
  lines+=("$key='$val'")
done < <(grep -E '^[A-Z][A-Z0-9_]*=' "$src/.env.example")
[ ${#missing[@]} -eq 0 ] ||
  die "create these CI/CD variables with environment scope $env (Protected; Masked if secret): ${missing[*]}"
tmp=$(mktemp "$dir/.env.XXXXXX")       # mode 600, owner deploy
printf '%s\n' "# Written by deploy/deploy.sh. Do not edit: change the CI/CD variable and deploy again." "${lines[@]}" >"$tmp"
mv -f "$tmp" "$dir/.env"

log "Copying compose files, postgres/ and kratos/ to $dir"
cp "$src/docker-compose.prod.yml" "$dir/compose.yaml"
cp "$src/docker-compose.staging.yml" "$dir/compose.staging.yaml"
for d in postgres kratos; do
  mkdir -p "$dir/$d"
  cp -rL "$src/$d/." "$dir/$d/"         # -L: postgres/roles.sql is a symlink to apps/api/db/roles.sql
done

cd "$dir"
docker compose config --quiet
log "Pulling images"
printf '%s' "$CI_DEPLOY_PASSWORD" | docker login --username "$CI_DEPLOY_USER" --password-stdin "$CI_REGISTRY" >/dev/null
docker compose pull --quiet --ignore-buildable

log "Migrating: msp migrate, then Kratos"
docker compose run --rm worker msp migrate
docker compose run --rm kratos-migrate        # `kratos migrate sql -e --yes`; `up` runs it again, a no-op

log "Starting containers"
docker compose up -d --build --wait --wait-timeout 300 --remove-orphans
curl -fsS --retry 10 --retry-delay 3 --retry-all-errors --max-time 10 "${MSP_PUBLIC_URL%/}/readyz"
echo
docker compose ps --format '{{.Service}} {{.Image}} {{.Status}}'
log "Deployed $env at $tag"
