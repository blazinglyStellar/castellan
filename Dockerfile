# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/castellan-api ./cmd/api/
RUN CGO_ENABLED=0 go build -o /bin/castellan-worker ./cmd/worker/

FROM alpine:3.21 AS base
RUN apk upgrade --no-cache

FROM base AS api
COPY --from=builder /bin/castellan-api /bin/castellan-api
EXPOSE 8080
CMD ["/bin/castellan-api"]

FROM base AS worker
COPY --from=builder /bin/castellan-worker /bin/castellan-worker
CMD ["/bin/castellan-worker"]
