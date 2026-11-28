#!/bin/sh
set -eu
umask 077

if [ -z "${PHOTO_ALBUM_JWT_SECRET:-}" ]; then
    key_file=/data/jwt-secret
    if [ ! -e "$key_file" ]; then
        key="$(od -An -N32 -tx1 /dev/urandom | tr -d '[:space:]')"
        [ "${#key}" -eq 64 ]
        printf '%s\n' "$key" > "$key_file.tmp"
        mv "$key_file.tmp" "$key_file"
    fi
    PHOTO_ALBUM_JWT_SECRET="$(cat "$key_file")"
    export PHOTO_ALBUM_JWT_SECRET
fi

exec /usr/local/bin/photo-album
