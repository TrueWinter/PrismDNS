#!/bin/bash

rm -r dist 2&>/dev/null

COMMIT=$(git rev-parse --short HEAD)

TAG=$(git describe --tags --abbrev=0 --candidates=100 2>/dev/null)
if [[ "$TAG" != "" ]]; then
  COMMIT="${TAG}-${COMMIT}"
fi

UNRESOLVED=$(git status -s)
if [[ "$UNRESOLVED" != "" ]]; then
  COMMIT="${COMMIT}-indev"
fi

build() {
  DIST_NAME="dist/dns-proxy-${GOOS}-${GOARCH}"
  if [ "$GOOS" == "windows" ]; then
    DIST_NAME=$DIST_NAME.exe
  fi
  go build -ldflags "-X main.Version=${COMMIT}" -o $DIST_NAME .
}

GOOS=windows GOARCH=amd64 build
GOOS=linux GOARCH=amd64 build
