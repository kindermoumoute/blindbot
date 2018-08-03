#!/usr/bin/env bash

VERSION=${VERSION:-latest}
IMAGE=${IMAGE:-quay.io/kindermoumoute/blindbot}
DOMAIN_NAME="example.org"
SLACK_KEY="xoxb-XXXXX..."
SLACK_OAUTH2_KEY="xoxp-XXXXXXXXXX..."
SLACK_MASTER="master.email@domain.com"
NAME=${NAME:-blindbot}

CURRENT_CONTAINER=$(docker ps --format "{{.ID}}" --filter "name=$NAME")
docker pull $IMAGE:$VERSION
docker kill $CURRENT_CONTAINER
# TODO: use docker-compose
ID=$(docker run --rm -d --name "$NAME" -v "$PWD"/db:/db -v "$PWD"/music:/music -v "$PWD"/cred:/cred -v '/etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt' -p 80:80 -p 443:443 -e SLACK_OAUTH2_KEY="$SLACK_OAUTH2_KEY" -e DOMAIN_NAME="$DOMAIN_NAME" -e SLACK_KEY="$SLACK_KEY" -e SLACK_MASTER="$SLACK_MASTER" $IMAGE:$VERSION)
docker logs $ID -f
