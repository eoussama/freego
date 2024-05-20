#!/bin/bash

tag="0.0.1"
project="freego"
username="eoussama"
image="$username/$project:$tag"

docker build -f ./docker/Dockerfile -t $image .
docker run -it --rm \
  -p 8080:8080 \
  -v "$(pwd)/src":/go/src/github.com/eoussama/freego \
  -v "$(pwd)/.env":/go/src/github.com/eoussama/freego/.env \
  -v "$(pwd)/scripts/smee.sh":/go/src/github.com/eoussama/freego/smee.sh \
  -v "$(pwd)/scripts/entrypoint.sh":/go/src/github.com/eoussama/freego/entrypoint.sh \
  $image