FROM golang:1.25.8-alpine AS vendor-snapshot
WORKDIR /snapshot
COPY go.mod go.sum ./
RUN go mod vendor

FROM scratch AS export
COPY --from=vendor-snapshot /snapshot/vendor /vendor