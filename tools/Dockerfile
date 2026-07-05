# syntax=docker/dockerfile:1
# Packages the local vendor/ directory as a minimal OCI image.
# No network access needed — vendor/ is already authoritative locally.
FROM scratch
COPY vendor/ /vendor/
