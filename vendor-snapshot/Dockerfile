FROM golang:1.25.8-alpine AS vendor-snapshot
WORKDIR /snapshot
COPY go.mod go.sum ./
RUN go mod vendor

FROM alpine:3.20
COPY --from=vendor-snapshot /snapshot/vendor /vendor