#!/bin/sh
# Runs from the nginx image entrypoint (/docker-entrypoint.d/) before nginx starts.
#
# /adminer/ is protected by HTTP Basic Auth. Credentials come from the runtime
# environment, so they are never baked into the image or committed to git.
# If they are missing we exit instead of serving an unprotected DB UI.
set -e

if [ -z "${ADMINER_USER:-}" ] || [ -z "${ADMINER_PASSWORD:-}" ]; then
    echo "ERROR: ADMINER_USER/ADMINER_PASSWORD wajib diisi; menolak start dengan /adminer terbuka." >&2
    exit 1
fi

# Must stay readable by the nginx worker user (nginx), otherwise auth fails 500.
htpasswd -bc /etc/nginx/.htpasswd "$ADMINER_USER" "$ADMINER_PASSWORD"
chmod 640 /etc/nginx/.htpasswd
chown root:nginx /etc/nginx/.htpasswd

echo "Adminer basic auth aktif untuk user '${ADMINER_USER}'."
