#!/usr/bin/env sh

set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
PLATFORM_DIR="$ROOT_DIR/cloud-infra"

usage() {
  cat <<'EOF'
Soul Room platform launcher

Usage:
  ./platform.sh <command> [service]

Commands:
  up                 Start the complete platform using local images
  build              Build local images, then start the platform
  down               Stop the platform and preserve persistent data
  refresh            Rebuild and recreate from the current source
  restart            Restart existing containers without rebuilding
  status             Show service and health status
  logs [service]     Follow all logs, or logs for one service
  doctor             Check Docker and validate the Compose configuration
  help               Show this help

Examples:
  ./platform.sh up
  ./platform.sh build
  ./platform.sh refresh
  ./platform.sh logs platform
EOF
}

require_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "Docker is not installed or is not available in PATH." >&2
    exit 1
  fi
  if ! docker compose version >/dev/null 2>&1; then
    echo "Docker Compose v2 is required. Install the 'docker compose' plugin." >&2
    exit 1
  fi
  if ! docker info >/dev/null 2>&1; then
    echo "Docker is installed, but the Docker engine is not running or cannot be reached." >&2
    exit 1
  fi
}

compose() {
  (cd "$PLATFORM_DIR" && docker compose "$@")
}

command=${1:-help}
if [ "$#" -gt 0 ]; then
  shift
fi

case "$command" in
  up)
    require_docker
    compose up --remove-orphans -d
    compose ps
    echo "Soul Room is available at http://localhost:3080"
    ;;
  build)
    require_docker
    compose build
    compose up --remove-orphans -d
    compose ps
    echo "Soul Room images were built and the platform is available at http://localhost:3080"
    ;;
  down)
    require_docker
    compose down --remove-orphans
    echo "Soul Room stopped. Persistent volumes were preserved."
    ;;
  refresh)
    require_docker
    compose up --build --force-recreate --remove-orphans -d
    compose ps
    echo "Soul Room was rebuilt and refreshed at http://localhost:3080"
    ;;
  restart)
    require_docker
    compose restart
    compose ps
    ;;
  status)
    require_docker
    compose ps
    ;;
  logs)
    require_docker
    if [ "$#" -gt 1 ]; then
      echo "Usage: ./platform.sh logs [service]" >&2
      exit 2
    fi
    if [ "$#" -eq 1 ]; then
      compose logs --tail=200 -f "$1"
    else
      compose logs --tail=200 -f
    fi
    ;;
  doctor)
    require_docker
    docker compose version
    compose config --quiet
    echo "Docker is ready and cloud-infra/compose.yaml is valid."
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    echo "Unknown command: $command" >&2
    usage >&2
    exit 2
    ;;
esac
