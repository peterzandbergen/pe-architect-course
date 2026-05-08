#!/usr/bin/env bash

# Build the Docker image
docker build -f operator.Dockerfile -t teams-operator:local .

# Load the image into your kind cluster so nodes can pull it
kind load docker-image teams-operator:local --name 5min-idp

kustomize build . | kubectl delete -f -

sleep 10

kustomize build . | kubectl apply -f -


