#!/bin/bash
set -e

# Match container user UID/GID to host user if FORGE_UID/FORGE_GID are set.
# This ensures files created in mounted volumes have correct host ownership.
if [ -n "$FORGE_UID" ] && [ -n "$FORGE_GID" ]; then
    current_uid=$(id -u user)
    current_gid=$(id -g user)

    if [ "$current_gid" != "$FORGE_GID" ]; then
        groupmod -g "$FORGE_GID" -o user 2>/dev/null || true
    fi
    if [ "$current_uid" != "$FORGE_UID" ]; then
        usermod -u "$FORGE_UID" -o user 2>/dev/null || true
    fi

    chown -R user:user /home/user 2>/dev/null || true
fi

exec runuser -u user -- "$@"
