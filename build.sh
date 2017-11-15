#!/bin/sh

IMAGE_ID=$(docker images ship-build -q)

if [[ -z "$IMAGE_ID" ]]; then
  echo "Build image doesn't exist, creating it"
  docker build -t ship-build .
fi

echo "Compiling project"

docker run --rm -it -v "$PWD":/go/src/github.com/SprintHive/ship ship-build go build -o ship .

echo "Done! The binary is called 'ship' in the current directory"
