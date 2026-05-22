#!/usr/bin/env bash
set -euo pipefail

LABEL="${1:-scheduled}"
DATA_ROOT="${DATA_ROOT:-/data/lumio}"
BACKUP_SCRIPT="${BACKUP_SCRIPT:-/opt/lumio/bin/backup.sh}"
LOCK_FILE="${LOCK_FILE:-${DATA_ROOT}/locks/lumio-disk-maintenance.lock}"
CLIPROXY_LOG_DIR="${CLIPROXY_LOG_DIR:-${DATA_ROOT}/cliproxy/logs}"
CLIPROXY_LOG_KEEP_DAYS="${CLIPROXY_LOG_KEEP_DAYS:-14}"
DOCKER_BUILDER_PRUNE_UNTIL="${DOCKER_BUILDER_PRUNE_UNTIL:-168h}"
DOCKER_CONTAINER_PRUNE_UNTIL="${DOCKER_CONTAINER_PRUNE_UNTIL:-24h}"
DOCKER_PRUNE_UNUSED_IMAGES="${DOCKER_PRUNE_UNUSED_IMAGES:-0}"
DOCKER_UNUSED_IMAGE_PRUNE_UNTIL="${DOCKER_UNUSED_IMAGE_PRUNE_UNTIL:-168h}"

if [[ ! "${CLIPROXY_LOG_KEEP_DAYS}" =~ ^[0-9]+$ ]]; then
  echo "CLIPROXY_LOG_KEEP_DAYS must be a non-negative integer." >&2
  exit 1
fi

mkdir -p "$(dirname "${LOCK_FILE}")"
touch "${LOCK_FILE}"
chmod 0660 "${LOCK_FILE}" 2>/dev/null || true
chgrp docker "${LOCK_FILE}" 2>/dev/null || true

exec 9>"${LOCK_FILE}"
flock -n 9 || {
  echo "Another disk maintenance run is active." >&2
  exit 1
}

if [[ -x "${BACKUP_SCRIPT}" ]]; then
  "${BACKUP_SCRIPT}" prune
else
  echo "Backup script not executable, skipping restic retention: ${BACKUP_SCRIPT}" >&2
fi

if [[ -d "${CLIPROXY_LOG_DIR}" ]]; then
  find "${CLIPROXY_LOG_DIR}" -type f \
    \( -name '*.log' -o -name '*.log.*' -o -name '*.out' \) \
    -mtime +"${CLIPROXY_LOG_KEEP_DAYS}" \
    -delete
fi

if command -v docker >/dev/null 2>&1; then
  docker container prune -f --filter "until=${DOCKER_CONTAINER_PRUNE_UNTIL}" >/dev/null || true
  docker image prune -f >/dev/null || true
  docker builder prune -af --filter "until=${DOCKER_BUILDER_PRUNE_UNTIL}" >/dev/null || true

  if [[ "${DOCKER_PRUNE_UNUSED_IMAGES}" == "1" ]]; then
    docker image prune -af --filter "until=${DOCKER_UNUSED_IMAGE_PRUNE_UNTIL}" >/dev/null || true
  fi
fi

df -h /data 2>/dev/null || true
echo "Disk maintenance complete: ${LABEL}"
