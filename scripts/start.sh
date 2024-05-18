#!/bin/bash

tag="0.0.1"
project="freego"
username="eoussama"
image="$username/$project:$tag"

docker build -f ./docker/Dockerfile -t $image .
docker run -it --rm \
  -v "$(pwd)/src":/go/src/github.com/eoussama/freego \
  $image