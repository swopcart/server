#!/bin/sh
set -e

SWOPCART_DATA="${SWOPCART_DATA:-/data}"
PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

# Create group and user with the requested UID/GID (idempotent)
if ! getent group swopcart > /dev/null 2>&1; then
    addgroup -g "$PGID" swopcart
fi
if ! getent passwd swopcart > /dev/null 2>&1; then
    adduser -D -u "$PUID" -G swopcart -s /sbin/nologin swopcart
fi

mkdir -p "$SWOPCART_DATA"

# Generate config on first launch
if [ ! -f "$SWOPCART_DATA/config.toml" ]; then
    cat > "$SWOPCART_DATA/config.toml" << EOF
[database]
conn = "${SWOPCART_DB_CONN:-postgres://swopcart:swopcart@localhost:5432/swopcart}"
EOF
fi

# Fix data directory ownership then drop to unprivileged user
chown -R swopcart:swopcart "$SWOPCART_DATA"
exec su-exec swopcart "$@"
