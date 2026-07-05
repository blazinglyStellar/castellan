# ---- Builder ----
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk upgrade --no-cache

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/castellan-api ./cmd/api/ && \
    CGO_ENABLED=0 GOOS=linux go build -o /bin/castellan-worker ./cmd/worker/

# ---- Runtime ----
FROM alpine:3.21

RUN apk upgrade --no-cache

COPY --from=builder /bin/castellan-api /bin/castellan-api
COPY --from=builder /bin/castellan-worker /bin/castellan-worker

EXPOSE 8080

CMD ["/bin/castellan-api"]
