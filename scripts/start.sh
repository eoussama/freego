#!/bin/bash

source .env

tag=$GLOBAL_VERSION
project="freego"
username="eoussama"
image="$username/$project:$tag"

docker build -f ./docker/Dockerfile -t $image .
docker run -it --rm \
  -p 8080:8080 \
  -v "$(pwd)":/go/src/github.com/eoussama/freego \
  $image