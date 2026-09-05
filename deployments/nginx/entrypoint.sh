#!/bin/sh
set -eu

# Earmarks the gateway↔api shared secret so the Go API can reject anything
# that does not arrive through this gateway (defense when the API network is
# exposed by misconfiguration). Empty APP_GATEWAY_TOKEN disables the header.
conf=/etc/nginx/conf.d/gateway-token.conf
if [ -n "${APP_GATEWAY_TOKEN:-}" ]; then
    # Variable so dynamically generated auth_request locations can forward
    # the same shared secret (a location with its own proxy_set_header
    # stops inheriting server-level ones).
    printf 'set $gateway_token "%s";\nproxy_set_header X-Gateway-Token $gateway_token;\n' "$APP_GATEWAY_TOKEN" > "$conf"
else
    printf '\n' > "$conf"
fi

# ── Upstream API host ────────────────────────────────────────────────────────
# nginx resolves the `upstream go_api` block at startup. Docker Compose
# provides a DNS name `api`; on Kubernetes the API container lives in the same
# Pod, so APP_UPSTREAM_API overrides to 127.0.0.1:8080. Written into the image
# (nginx.conf is a read-only bind-mount, not editable at runtime).
printf 'upstream go_api {\n    server %s;\n    keepalive 32;\n}\n' \
    "${APP_UPSTREAM_API:-api:8080}" > /etc/nginx/upstream.conf

# ── Registered-API registry (dynamic nginx includes) ───────────────────────
# The Go API writes locations/zones.conf into the shared volume. Ensure the
# runtime layout exists so the static nginx.conf includes resolve cleanly.
mkdir -p /etc/nginx/registry/locations /etc/nginx/registry/logs
[ -f /etc/nginx/registry/zones.conf ] || touch /etc/nginx/registry/zones.conf

# Watch the registry and hot-reload nginx whenever its content changes
# (registered APIs appear/disappear without restarting the gateway). The Go
# API only touches files; this side-channel triggers the reload.
watch_registry() {
    last=""
    while :; do
        sleep 1
        cur=$(find /etc/nginx/registry -type f -name '*.conf' -exec sha1sum {} \; 2>/dev/null | sha1sum)
        if [ "$cur" != "$last" ]; then
            last="$cur"
            nginx -s reload 2>/dev/null || true
        fi
    done
}
watch_registry &

exec nginx -g 'daemon off;'