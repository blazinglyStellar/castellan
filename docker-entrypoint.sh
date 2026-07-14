#!/bin/sh
set -e

/bin/goose -dir /migrations up

exec "$@"
