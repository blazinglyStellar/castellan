# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/castellan-api ./cmd/api/
RUN CGO_ENABLED=0 go build -o /bin/castellan-worker ./cmd/worker/

FROM golang:1.26-alpine AS goose-builder
RUN go install github.com/pressly/goose/v3/cmd/goose@latest

FROM alpine:3.21 AS migration
COPY --from=goose-builder /go/bin/goose /bin/goose
COPY migrations /migrations
CMD ["/bin/goose", "-dir", "/migrations", "up"]

FROM alpine:3.21 AS base
RUN apk upgrade --no-cache
RUN apk add --no-cache wget

FROM base AS api
COPY --from=builder /bin/castellan-api /bin/castellan-api
COPY --from=goose-builder /bin/goose /bin/goose
COPY migrations /migrations
COPY docker-entrypoint.sh /docker-entrypoint.sh
EXPOSE 8080
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/bin/castellan-api"]

FROM base AS worker
COPY --from=builder /bin/castellan-worker /bin/castellan-worker
CMD ["/bin/castellan-worker"]
