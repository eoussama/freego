#!/bin/bash
# Builds the development image and opens a shell in it, with the project
# mounted. Run it from the project root.
set -euo pipefail

if [ -f .env ]; then
  # shellcheck disable=SC1091
  source .env
fi

tag="${GLOBAL_VERSION:-latest}"
image="eoussama/freego:$tag"
port="${FREEGO_WEBHOOK_PORT:-8080}"

docker build -f ./docker/Dockerfile -t "$image" .
# Named so that a second shell can join it: docker exec -it freego-dev bash
docker run -it --rm \
  --name freego-dev \
  -p "$port:$port" \
  -v "$(pwd)":/go/src/github.com/eoussama/freego \
  "$image"
