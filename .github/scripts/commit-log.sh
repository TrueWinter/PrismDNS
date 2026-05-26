#!/bin/bash

PREV_TAG=$(git describe --tags --abbrev=0 --candidates=100 HEAD^ 2>/dev/null)
TAG=$(git describe --tags --abbrev=0 --candidates=100 2>/dev/null)
REV_RANGE="$PREV_TAG...$TAG"
if [[ "$PREV_TAG" == "" || "$PREV_TAG" == "$TAG" ]]; then
  REV_RANGE="$TAG"
fi

git --no-pager log --format="%s (%h)" $REV_RANGE
