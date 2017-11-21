#!/bin/sh

IMAGE_ID=$(docker images ship-build -q)

if test -z "$IMAGE_ID"; then
  echo "Build image doesn't exist, creating it"
  docker build -t ship-build .
fi

GOOS=${GOOS:-darwin}
GOARCH=${GOARCH:-amd64}

echo "Starting build container..."
docker run --name ship-yard -e CGO_ENABLED=0 -e GOOS=$GOOS -e GOARCH=$GOARCH -dt -v "$PWD":/go/src/github.com/SprintHive/ship ship-build cat
echo "Downloading ship dependencies"
docker exec ship-yard dep ensure -update
echo "Compiling project"
docker exec ship-yard go build -a -installsuffix cgo -o ship .
docker rm -f ship-yard

echo "Done! The binary is called 'ship' in the current directory"
