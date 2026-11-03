#!/bin/bash

# Compute tag
GOSUM_SHA=$(sha256sum go.sum | awk '{print $1}')

# Login
echo $GH_PAT | oras login ghcr.io -u blanketops --password-stdin

# Push vendor snapshot
oras push \
  ghcr.io/blanketops/environments/tools:${GOSUM_SHA} \
  ./vendor
