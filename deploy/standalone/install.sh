#!/usr/bin/env bash
# Install APFS as a systemd-managed Docker Compose stack on Ubuntu/Debian.
#
#   curl -fsSL https://raw.githubusercontent.com/apfs-io/apfs/main/deploy/standalone/install.sh | sudo bash
#
# Optional environment:
#   APFS_REPO    GitHub owner/name          (default: apfs-io/apfs)
#   APFS_REF     git ref for config files   (default: main)
#   APFS_PREFIX  install directory          (default: /opt/apfs)
#   APFS_IMAGE   override the APFS image    (written into compose env)
#   APFS_SKIP_START=1   install only, do not enable/start systemd
set -euo pipefail

APFS_REPO="${APFS_REPO:-apfs-io/apfs}"
APFS_REF="${APFS_REF:-main}"
APFS_PREFIX="${APFS_PREFIX:-/opt/apfs}"
APFS_SKIP_START="${APFS_SKIP_START:-0}"
RAW_BASE="https://raw.githubusercontent.com/${APFS_REPO}/${APFS_REF}/deploy/standalone"
UNIT_LINK="/etc/systemd/system/apfs.service"

log()  { printf '==> %s\n' "$*"; }
die()  { printf 'error: %s\n' "$*" >&2; exit 1; }

require_root() {
  [[ "$(id -u)" -eq 0 ]] || die "run as root (sudo)"
}

require_apt() {
  [[ -r /etc/os-release ]] || die "cannot read /etc/os-release"
  # shellcheck disable=SC1091
  . /etc/os-release
  case "${ID:-}" in
    ubuntu|debian) ;;
    *) die "unsupported OS '${ID:-unknown}': need Ubuntu or Debian" ;;
  esac
  command -v apt-get >/dev/null || die "apt-get not found"
}

apt_install() {
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -y
  apt-get install -y --no-install-recommends ca-certificates curl gnupg
}

docker_compose_ok() {
  docker compose version >/dev/null 2>&1
}

install_docker() {
  if command -v docker >/dev/null && docker_compose_ok; then
    log "Docker Compose already available"
    return
  fi

  log "Installing Docker Engine and Compose plugin"
  # shellcheck disable=SC1091
  . /etc/os-release
  local distro="${ID}"
  local codename="${VERSION_CODENAME:-}"
  [[ -n "${codename}" ]] || die "VERSION_CODENAME missing in /etc/os-release"

  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL "https://download.docker.com/linux/${distro}/gpg" -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc
  cat > /etc/apt/sources.list.d/docker.list <<EOF
deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/${distro} ${codename} stable
EOF
  apt-get update -y
  apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  docker_compose_ok || die "docker compose plugin is not usable after install"
}

enable_docker() {
  systemctl enable --now docker.service
}

docker_bin() {
  if [[ -x /usr/bin/docker ]] && /usr/bin/docker compose version >/dev/null 2>&1; then
    printf '%s' /usr/bin/docker
    return
  fi
  command -v docker
}

compose_cmd() {
  local bin
  bin="$(docker_bin)"
  if "${bin}" compose version >/dev/null 2>&1; then
    printf '%s compose' "${bin}"
    return
  fi
  if command -v docker-compose >/dev/null; then
    command -v docker-compose
    return
  fi
  die "neither 'docker compose' nor 'docker-compose' is available"
}

download() {
  local name="$1" dest="$2"
  local url="${RAW_BASE}/${name}"
  log "Fetching ${url}"
  curl -fsSL "${url}" -o "${dest}" || die "failed to download ${url}"
}

install_files() {
  mkdir -p \
    "${APFS_PREFIX}/data/storage" \
    "${APFS_PREFIX}/data/meta" \
    "${APFS_PREFIX}/data/redis" \
    "${APFS_PREFIX}/workflows"

  download docker-compose.yaml "${APFS_PREFIX}/docker-compose.yaml"
  download apfs.service        "${APFS_PREFIX}/apfs.service"
  download install.sh          "${APFS_PREFIX}/install.sh"
  chmod 0755 "${APFS_PREFIX}/install.sh"

  download apfs.env "${APFS_PREFIX}/apfs.env.example"
  if [[ -f "${APFS_PREFIX}/apfs.env" ]]; then
    log "Keeping existing ${APFS_PREFIX}/apfs.env (latest defaults in apfs.env.example)"
  else
    cp "${APFS_PREFIX}/apfs.env.example" "${APFS_PREFIX}/apfs.env"
    log "Wrote ${APFS_PREFIX}/apfs.env from repository defaults"
  fi

  if [[ -n "${APFS_IMAGE:-}" ]]; then
    # Compose interpolates ${APFS_IMAGE} from project .env, not from env_file.
    printf 'APFS_IMAGE=%s\n' "${APFS_IMAGE}" > "${APFS_PREFIX}/.env"
  fi
}

patch_unit() {
  local unit="${APFS_PREFIX}/apfs.service"
  local bin compose start stop
  bin="$(docker_bin)"
  compose="$(compose_cmd)"

  sed -i "s|^WorkingDirectory=.*|WorkingDirectory=${APFS_PREFIX}|" "${unit}"

  if [[ "${compose}" == *' compose' ]]; then
    start="${bin} compose up -d --remove-orphans"
    stop="${bin} compose down"
  else
    start="${compose} up -d --remove-orphans"
    stop="${compose} down"
  fi
  sed -i "s|^ExecStart=.*|ExecStart=${start}|" "${unit}"
  sed -i "s|^ExecStop=.*|ExecStop=${stop}|" "${unit}"
  sed -i "s|^ExecReload=.*|ExecReload=${start}|" "${unit}"
}

install_unit() {
  patch_unit
  ln -sfn "${APFS_PREFIX}/apfs.service" "${UNIT_LINK}"
  systemctl daemon-reload
}

pull_images() {
  log "Pulling images"
  # shellcheck disable=SC2086
  if ! (cd "${APFS_PREFIX}" && $(compose_cmd) pull); then
    cat >&2 <<EOF
error: failed to pull images.
If ghcr.io/apfs-io/apfs is private, authenticate first:

  echo "\$GITHUB_TOKEN" | docker login ghcr.io -u USERNAME --password-stdin

Then re-run this script.
EOF
    exit 1
  fi
}

start_stack() {
  if [[ "${APFS_SKIP_START}" == "1" ]]; then
    log "Skipping start (APFS_SKIP_START=1). Enable later with: systemctl enable --now apfs"
    return
  fi
  log "Enabling and starting apfs.service"
  systemctl enable --now apfs.service
  systemctl --no-pager --full status apfs.service || true
}

main() {
  require_root
  require_apt
  apt_install
  install_docker
  enable_docker
  install_files
  install_unit
  pull_images
  start_stack
  log "APFS installed at ${APFS_PREFIX}"
  log "HTTP  http://localhost:8080   gRPC  localhost:8081"
  log "Config: ${APFS_PREFIX}/apfs.env"
}

main "$@"
