#!/bin/bash

# Compute tag
GOSUM_SHA=$(sha256sum go.sum | awk '{print $1}')

# Login
echo $GH_PAT | oras login ghcr.io -u ntlaletsi70 --password-stdin

# Push vendor snapshot
oras push \
  ghcr.io/ntlaletsi70/blanketops-environments-controller/vendor-snapshot:${GOSUM_SHA} \
  ./vendor
