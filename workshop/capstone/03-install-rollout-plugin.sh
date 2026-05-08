#!/usr/bin/env bash

# Download the exec to /tmp
(
	cd /tmp
	curl -LO https://github.com/argoproj/argo-rollouts/releases/latest/download/kubectl-argo-rollouts-linux-amd64
	chmod +x /tmp/kubectl-argo-rollouts-linux-amd64
	mv /tmp/kubectl-argo-rollouts-linux-amd64 ~/.local/bin/kubectl-argo-rollouts
)

