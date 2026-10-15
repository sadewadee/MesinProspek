#!/bin/sh
# Off-site copy of the local pgBackRest repo and the GitLab backups (docs/06-ci-cd.md §8, D-39).
# Run on the host by root cron /etc/cron.d/msp-offsite every 5 minutes, not by GitLab, so it keeps
# running while GitLab is down. Success time per part goes to node-exporter's textfile collector (09 AL-12).
export RCLONE_CONFIG=/opt/ops/rclone.conf
R=/opt/msp/pgbackrest G=/opt/ops/gitlab-backups T=/opt/ops/textfile   # T -> node-exporter --collector.textfile.directory
ok() { echo "msp_offsite_last_success_timestamp_seconds{part=\"$1\"} $(date +%s)" > "$T/.$1" && mv "$T/.$1" "$T/offsite_$1.prom"; }

# An empty source (new server, deleted repo) is never synced: sync also deletes at the destination
if [ ! -s "$R/archive/msp/archive.info" ] || [ ! -s "$R/backup/msp/backup.info" ]; then
  echo "local pgBackRest repo $R is empty: nothing synced (09 AL-12)" >&2
  exit 1
fi

# 1. WAL: small, every run, never waits for the big upload in step 2 (RPO)
flock -n /run/lock/msp-offsite-wal rclone sync "$R/archive" offsite:msp/pgbackrest/archive && ok wal

# 2. pgBackRest backups + GitLab; skipped while the previous run is still uploading (e.g. weekly full)
if ! ls "$G"/*_gitlab_backup.tar >/dev/null 2>&1; then
  echo "no GitLab backup in $G yet: backups not synced (09 AL-12)" >&2
  exit 1
fi
flock -n /run/lock/msp-offsite-backup sh -ec "
  rclone sync '$R/backup' offsite:msp/pgbackrest/backup
  rclone sync '$G' offsite:msp/gitlab-backups
  # rclone sync '$R' b2:<bucket>/msp/pgbackrest    # second copy on B2: DISABLED until legal answers Q-19
" && ok backup
