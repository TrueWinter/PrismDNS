#!/bin/bash

VERSION=""

COMMIT=$(git rev-parse --short HEAD)

TAG=$(git describe --tags --abbrev=0 --candidates=100 2>/dev/null)
TAG_COMMIT=$(git rev-parse --short $TAG 2>/dev/null)
if [[ "$TAG" != "" ]]; then
  if [[ "$COMMIT" == "$TAG_COMMIT" ]]; then
    VERSION="$TAG"
  else
    VERSION="$TAG-$COMMIT"
  fi
else
  VERSION="$COMMIT"
fi

echo $VERSION
