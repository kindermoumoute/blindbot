#!/usr/bin/env bash

set -e
set -x

if [[ ("$TRAVIS_BRANCH" == "master" && "$TRAVIS_PULL_REQUEST_BRANCH" == "") || "$TRAVIS_TAG" != "" ]]; then
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o blindbot .
  docker login -u="kindermoumoute" -p="$DOCKER_PASS" quay.io
  REPO="quay.io/kindermoumoute/blindbot"
  docker build -f Dockerfile -t $REPO:latest .
  if [[ "$TRAVIS_TAG" != "" ]]; then
    docker tag $REPO:latest $REPO:$TRAVIS_TAG
  fi
  docker push $REPO
fi
