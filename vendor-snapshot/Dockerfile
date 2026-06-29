ARG GO_VERSION=1.26
FROM golang:${GO_VERSION}-alpine AS vendor-snapshot

WORKDIR /snapshot

# git is not included in the golang alpine image
RUN apk add --no-cache git

COPY go.mod go.sum ./

RUN --mount=type=secret,id=GH_PAT \
    git config --global url."https://$(cat /run/secrets/GH_PAT)@github.com/".insteadOf "https://github.com/" && \
    go mod vendor

FROM scratch AS export
COPY --from=vendor-snapshot /snapshot/vendor /vendor