# syntax=docker/dockerfile:1
FROM alpine:3.21 AS base
RUN apk upgrade --no-cache

FROM base AS api
COPY bin/castellan-api /bin/castellan-api
EXPOSE 8080
CMD ["/bin/castellan-api"]

FROM base AS worker
COPY bin/castellan-worker /bin/castellan-worker
CMD ["/bin/castellan-worker"]
