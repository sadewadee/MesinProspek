#!/bin/sh
# Entrypoint of the alertmanager container (busybox sh + awk in the image). Alertmanager cannot read
# environment variables in its config, so this renders alertmanager.yml (template, no secrets) with the
# ${MSP_...} values from the container environment, then starts Alertmanager on the rendered file.
set -eu
umask 077
awk '{
  out = ""; s = $0
  while (match(s, /\$\{[A-Z0-9_]+\}/)) {
    out = out substr(s, 1, RSTART - 1) ENVIRON[substr(s, RSTART + 2, RLENGTH - 3)]
    s = substr(s, RSTART + RLENGTH)
  }
  print out s
}' /etc/alertmanager/alertmanager.yml > /alertmanager/config.yml
exec /bin/alertmanager --config.file=/alertmanager/config.yml --storage.path=/alertmanager \
  --cluster.listen-address=   # single node: no gossip port
