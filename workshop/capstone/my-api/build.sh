#!/usr/bin/env bash

docker build -t my-api:local-v1 src
docker tag my-api:local-v1 docker.io/peterzandbergen/pe-arch-my-api:v1
docker push docker.io/peterzandbergen/pe-arch-my-api:v1
