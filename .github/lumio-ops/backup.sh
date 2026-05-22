#!/usr/bin/env bash
set -euo pipefail
umask 007

ACTION="${1:-scheduled}"
LABEL="${ACTION}"
PRUNE_ONLY=0
if [[ "${ACTION}" == "prune" ]]; then
  LABEL="retention"
  PRUNE_ONLY=1
fi
BACKUP_ENV="${BACKUP_ENV:-/data/lumio/backup.env}"
COMPOSE_ENV="${COMPOSE_ENV:-/data/lumio/compose/.env.production}"
STAGING_DIR="${STAGING_DIR:-/data/lumio/backups/staging}"
LOCK_FILE="${LOCK_FILE:-/data/lumio/locks/lumio-backup.lock}"
CPA_DATA_DIR="${CPA_DATA_DIR:-/data/lumio/cpa-manager/data}"
RESTIC_CACHE_DIR="${RESTIC_CACHE_DIR:-/data/lumio/backups/restic-cache}"
STAGING_KEEP_DAYS="${STAGING_KEEP_DAYS:-1}"
STAGING_KEEP_COUNT="${STAGING_KEEP_COUNT:-1}"
BACKUP_MIN_FREE_MB="${BACKUP_MIN_FREE_MB:-8192}"
BACKUP_NICE_LEVEL="${BACKUP_NICE_LEVEL:-10}"
BACKUP_IONICE_CLASS="${BACKUP_IONICE_CLASS:-2}"
BACKUP_IONICE_LEVEL="${BACKUP_IONICE_LEVEL:-7}"
RESTIC_PRUNE_INTERVAL_HOURS="${RESTIC_PRUNE_INTERVAL_HOURS:-0}"
RESTIC_PRUNE_MARKER="${RESTIC_PRUNE_MARKER:-${STAGING_DIR}/.restic-prune-last}"

if [[ ! -f "${BACKUP_ENV}" ]]; then
  echo "Missing backup env file: ${BACKUP_ENV}" >&2
  exit 1
fi

if [[ ! -f "${COMPOSE_ENV}" ]]; then
  echo "Missing compose env file: ${COMPOSE_ENV}" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
. "${COMPOSE_ENV}"
# shellcheck disable=SC1090
. "${BACKUP_ENV}"
set +a

: "${RESTIC_REPOSITORY:?RESTIC_REPOSITORY is required}"
: "${RESTIC_PASSWORD_FILE:?RESTIC_PASSWORD_FILE is required}"

RESTIC_CACHE_BASE="${RESTIC_CACHE_DIR%/}"
RESTIC_CACHE_DIR="${RESTIC_CACHE_BASE}/$(id -un)"

if [[ ! "${STAGING_KEEP_DAYS}" =~ ^[0-9]+$ ]]; then
  echo "STAGING_KEEP_DAYS must be a non-negative integer." >&2
  exit 1
fi

if [[ ! "${STAGING_KEEP_COUNT}" =~ ^[0-9]+$ ]]; then
  echo "STAGING_KEEP_COUNT must be a non-negative integer." >&2
  exit 1
fi

if [[ ! "${BACKUP_MIN_FREE_MB}" =~ ^[0-9]+$ ]]; then
  echo "BACKUP_MIN_FREE_MB must be a non-negative integer." >&2
  exit 1
fi

if [[ ! "${BACKUP_NICE_LEVEL}" =~ ^-?[0-9]+$ ]]; then
  echo "BACKUP_NICE_LEVEL must be an integer." >&2
  exit 1
fi

if [[ ! "${BACKUP_IONICE_CLASS}" =~ ^[0-9]+$ || ! "${BACKUP_IONICE_LEVEL}" =~ ^[0-9]+$ ]]; then
  echo "BACKUP_IONICE_CLASS and BACKUP_IONICE_LEVEL must be non-negative integers." >&2
  exit 1
fi

if [[ ! "${RESTIC_PRUNE_INTERVAL_HOURS}" =~ ^[0-9]+$ ]]; then
  echo "RESTIC_PRUNE_INTERVAL_HOURS must be a non-negative integer." >&2
  exit 1
fi

mkdir -p "${STAGING_DIR}" "${RESTIC_CACHE_DIR}"

lock_dir="$(dirname "${LOCK_FILE}")"
mkdir -p "${lock_dir}"
touch "${LOCK_FILE}"
chmod 0660 "${LOCK_FILE}" 2>/dev/null || true
chgrp docker "${LOCK_FILE}" 2>/dev/null || true

exec 9>"${LOCK_FILE}"
flock -n 9 || {
  echo "Another backup is running." >&2
  exit 1
}

cleanup_staging_by_count() {
  local pattern="$1"
  local keep_count="$2"
  local index=0
  local entry
  local path

  if [[ "${keep_count}" -eq 0 ]]; then
    return
  fi

  while IFS= read -r -d '' entry; do
    path="${entry#* }"
    index=$((index + 1))
    if [[ "${index}" -gt "${keep_count}" ]]; then
      rm -f -- "${path}"
    fi
  done < <(find "${STAGING_DIR}" -maxdepth 1 -name "${pattern}" -type f -printf '%T@ %p\0' | sort -z -rn)
}

cleanup_staging() {
  find "${STAGING_DIR}" -maxdepth 1 -type f -size 0 ! -path "${RESTIC_PRUNE_MARKER}" -delete
  find "${STAGING_DIR}" -name 'sub2api-postgres-*.dump' -type f -mtime +"${STAGING_KEEP_DAYS}" -delete
  find "${STAGING_DIR}" -name 'cpa-manager-usage-*.sqlite' -type f -mtime +"${STAGING_KEEP_DAYS}" -delete
  cleanup_staging_by_count 'sub2api-postgres-*.dump' "${STAGING_KEEP_COUNT}"
  cleanup_staging_by_count 'cpa-manager-usage-*.sqlite' "${STAGING_KEEP_COUNT}"
}

require_free_space() {
  local path="$1"
  local min_mb="$2"
  local available_mb

  if [[ "${min_mb}" -eq 0 ]]; then
    return
  fi

  available_mb="$(df -Pm "${path}" | awk 'NR == 2 { print $4 }')"
  if [[ -z "${available_mb}" || "${available_mb}" -lt "${min_mb}" ]]; then
    echo "Not enough free space on ${path}: available=${available_mb:-unknown}MB required=${min_mb}MB" >&2
    exit 1
  fi
}

low_priority() {
  local cmd=(nice -n "${BACKUP_NICE_LEVEL}")

  if command -v ionice >/dev/null 2>&1; then
    cmd=(ionice -c "${BACKUP_IONICE_CLASS}" -n "${BACKUP_IONICE_LEVEL}" "${cmd[@]}")
  fi

  "${cmd[@]}" "$@"
}

should_prune_restic() {
  local interval_hours="$1"

  if [[ "${RESTIC_PRUNE:-auto}" == "always" || "${RESTIC_PRUNE:-auto}" == "1" ]]; then
    return 0
  fi
  if [[ "${RESTIC_PRUNE:-auto}" == "never" || "${RESTIC_PRUNE:-auto}" == "0" ]]; then
    return 1
  fi
  if [[ "${interval_hours}" -eq 0 ]]; then
    return 0
  fi
  if [[ ! -f "${RESTIC_PRUNE_MARKER}" ]]; then
    return 0
  fi

  find "${RESTIC_PRUNE_MARKER}" -mmin +"$((interval_hours * 60))" | grep -q .
}

unlock_stale_restic_locks() {
  low_priority restic unlock >/dev/null
}

apply_restic_retention() {
  local force_prune="${1:-0}"
  local retention_configured=0
  local -a forget_args

  forget_args=(--tag lumio --group-by host)

  if [[ -n "${RESTIC_KEEP_WITHIN:-}" ]]; then
    forget_args+=(--keep-within "${RESTIC_KEEP_WITHIN}")
    retention_configured=1
  fi

  if [[ -n "${RESTIC_KEEP_HOURLY+x}" && -n "${RESTIC_KEEP_HOURLY}" ]]; then
    forget_args+=(--keep-hourly "${RESTIC_KEEP_HOURLY}")
    retention_configured=1
  fi

  if [[ -n "${RESTIC_KEEP_DAILY+x}" && -n "${RESTIC_KEEP_DAILY}" ]]; then
    forget_args+=(--keep-daily "${RESTIC_KEEP_DAILY}")
    retention_configured=1
  fi

  if [[ -n "${RESTIC_KEEP_WEEKLY+x}" && -n "${RESTIC_KEEP_WEEKLY}" ]]; then
    forget_args+=(--keep-weekly "${RESTIC_KEEP_WEEKLY}")
    retention_configured=1
  fi

  if [[ -n "${RESTIC_KEEP_MONTHLY+x}" && -n "${RESTIC_KEEP_MONTHLY}" ]]; then
    forget_args+=(--keep-monthly "${RESTIC_KEEP_MONTHLY}")
    retention_configured=1
  fi

  if [[ "${retention_configured}" -eq 0 ]]; then
    forget_args+=(--keep-hourly 6 --keep-daily 3)
  fi

  if [[ "${force_prune}" == "1" ]] || should_prune_restic "${RESTIC_PRUNE_INTERVAL_HOURS}"; then
    low_priority restic forget \
      "${forget_args[@]}" \
      --prune
    touch "${RESTIC_PRUNE_MARKER}"
  else
    low_priority restic forget "${forget_args[@]}"
  fi
}

cleanup_staging

timestamp="$(date -u +%Y%m%d-%H%M%S)"
dump_name="sub2api-postgres-${timestamp}-${LABEL}.dump"
dump_path="${STAGING_DIR}/${dump_name}"
cpa_sqlite_backup=""

export RESTIC_REPOSITORY RESTIC_PASSWORD_FILE RESTIC_CACHE_DIR

if [[ "${PRUNE_ONLY}" -eq 1 ]]; then
  if restic snapshots --no-lock >/dev/null 2>&1; then
    unlock_stale_restic_locks
    apply_restic_retention 1
  fi
  cleanup_staging
  echo "Backup retention complete"
  exit 0
fi

require_free_space "${STAGING_DIR}" "${BACKUP_MIN_FREE_MB}"

if ! restic snapshots --no-lock >/dev/null 2>&1; then
  restic init
fi
unlock_stale_restic_locks

low_priority docker exec \
  -e PGPASSWORD="${POSTGRES_PASSWORD}" \
  -e PGUSER="${POSTGRES_USER:-sub2api}" \
  -e PGDATABASE="${POSTGRES_DB:-sub2api}" \
  -e BACKUP_NICE_LEVEL="${BACKUP_NICE_LEVEL}" \
  lumio-postgres \
  sh -c 'if command -v nice >/dev/null 2>&1; then exec nice -n "${BACKUP_NICE_LEVEL}" pg_dump -U "${PGUSER}" -d "${PGDATABASE}" -Fc; fi; exec pg_dump -U "${PGUSER}" -d "${PGDATABASE}" -Fc' \
  >"${dump_path}"
chmod 0640 "${dump_path}"

docker exec lumio-redis sh -c 'redis-cli ${REDISCLI_AUTH:+-a "$REDISCLI_AUTH"} BGSAVE || true' >/dev/null 2>&1 || true

if [[ -f "${CPA_DATA_DIR}/usage.sqlite" ]]; then
  if ! command -v sqlite3 >/dev/null 2>&1; then
    echo "sqlite3 is required to create a consistent CPA usage backup." >&2
    exit 1
  fi
  cpa_sqlite_backup="${STAGING_DIR}/cpa-manager-usage-${timestamp}-${LABEL}.sqlite"
  low_priority sqlite3 "${CPA_DATA_DIR}/usage.sqlite" ".backup '${cpa_sqlite_backup}'"
  chmod 0640 "${cpa_sqlite_backup}"
fi

backup_paths=(
  "${dump_path}"
  /data/lumio/compose
  /data/lumio/sub2api-data
  /data/lumio/cliproxy
  /data/lumio/cpa-manager
  /data/lumio/redis
)

if [[ -n "${cpa_sqlite_backup}" ]]; then
  backup_paths+=("${cpa_sqlite_backup}")
fi

low_priority restic backup \
  --tag lumio \
  --tag "${LABEL}" \
  "${backup_paths[@]}"

unlock_stale_restic_locks
apply_restic_retention

cleanup_staging

echo "Backup complete: ${dump_name}"
