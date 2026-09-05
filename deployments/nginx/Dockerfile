# Gateway image: nginx + gateway hardening config + the management console
# build (serves /usr/share/nginx/html → /admin/). Built with context = repo
# root so COPY paths are relative to the repository.
#
#   docker build -f deployments/nginx/Dockerfile -t go-enterprise-gateway .
FROM nginx:1.27-alpine

# Gateway-specific hardening (WAF-ish filters, rate limits, headers,
# /admin/ console, registry includes) — see deployments/nginx/nginx.conf.
COPY deployments/nginx/nginx.conf /etc/nginx/nginx.conf

# Entrypoint: injects X-Gateway-Token from APP_GATEWAY_TOKEN, bootstraps the
# shared registry dirs and runs the config hot-reload watcher.
COPY deployments/nginx/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Built management console (shell + micro frontends). Build first:
#   make build-frontend
COPY frontend/dist /usr/share/nginx/html

EXPOSE 80
ENTRYPOINT ["/entrypoint.sh"]