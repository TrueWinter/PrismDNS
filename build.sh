#!/bin/bash

rm -r dist 2&>/dev/null

VERSION=$(./.github/scripts/version.sh)

build() {
  DIST_NAME="dist/PrismDNS-${GOOS}-${GOARCH}"
  if [ "$GOOS" == "windows" ]; then
    DIST_NAME=$DIST_NAME.exe
  fi
  echo "Building $DIST_NAME $VERSION"
  go build -ldflags "-X main.Version=${VERSION} -s" -o $DIST_NAME .
}

GOOS=windows GOARCH=amd64 build
GOOS=linux GOARCH=amd64 build
