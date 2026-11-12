#!/bin/sh
# Runs from the nginx image entrypoint (/docker-entrypoint.d/) before nginx starts.
#
# TLS terminates here, so no separate host proxy is needed.
#   * Real certificates: put fullchain.pem + privkey.pem in ./certs on the host
#     (mounted read-only at /etc/nginx/certs).
#   * If they are missing, a self-signed pair is generated into /etc/nginx/tls so
#     the stack still boots (browsers will warn — not for production).
set -e

SRC=/etc/nginx/certs
DST=/etc/nginx/tls
NAME="${EDGE_SERVER_NAME:-localhost}"

mkdir -p "$DST"

if [ -s "$SRC/fullchain.pem" ] && [ -s "$SRC/privkey.pem" ]; then
    cp "$SRC/fullchain.pem" "$DST/fullchain.pem"
    cp "$SRC/privkey.pem" "$DST/privkey.pem"
    echo "TLS: memakai sertifikat dari $SRC"
else
    echo "TLS: sertifikat tidak ada di $SRC — membuat self-signed untuk '$NAME' (jangan dipakai di produksi)." >&2
    openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
        -keyout "$DST/privkey.pem" -out "$DST/fullchain.pem" \
        -subj "/CN=$NAME" -addext "subjectAltName=DNS:$NAME" >/dev/null 2>&1
fi

chmod 600 "$DST/privkey.pem"
chmod 644 "$DST/fullchain.pem"
