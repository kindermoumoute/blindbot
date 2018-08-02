#!/usr/bin/env bash

VERSION=${VERSION:-latest}
IMAGE=${IMAGE:-quay.io/kindermoumoute/blindbot}
DOMAIN_NAME="example.org"
SLACK_KEY="XXXXX..."
SLACK_MASTER="master.email@domain.com"

CURRENT_CONTAINER=$(docker ps --format "{{.ID}}" --filter "ancestor=$IMAGE")
docker pull $IMAGE:$VERSION
docker kill $CURRENT_CONTAINER
ID=$(docker run --rm -d --name blindbot -v $PWD/db:/db -v $PWD/music:/music -v $PWD/cred:/cred -v '/etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt' -p 80:80 -p 443:443 -e DOMAIN_NAME="$DOMAIN_NAME" -e SLACK_KEY="$SLACK_KEY" -e SLACK_MASTER="$SLACK_MASTER" $IMAGE:$VERSION)
docker logs $ID -f
