# syntax=docker/dockerfile:1
FROM golang:1.26.10-bookworm AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
# Keep CGO enabled for the SQLite driver.
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    mkdir -p /out && CGO_ENABLED=1 go build -trimpath -o /out/api . && \
    CGO_ENABLED=1 go build -trimpath -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=1 go build -trimpath -o /out/worker ./cmd/worker && \
    CGO_ENABLED=1 go build -trimpath -o /out/seed-admin ./cmd/seed-admin

FROM node:24-bookworm-slim AS web-build
WORKDIR /src
COPY package*.json ./
COPY clients/web ./clients/web
COPY modules/auth/resources ./modules/auth/resources
COPY modules/tenancy/resources ./modules/tenancy/resources
COPY resources ./resources
COPY clients/mobile/package.json ./clients/mobile/package.json
COPY packages/api-client ./packages/api-client
RUN --mount=type=cache,target=/root/.npm \
    if [ -f package-lock.json ]; then npm ci --workspace @edupilot/web --workspace @edupilot/api-client; \
    else npm install --workspace @edupilot/web --workspace @edupilot/api-client; fi
RUN npm run build --workspace @edupilot/web

FROM debian:bookworm-slim AS backend
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd --system --gid 10001 edupilot && useradd --system --uid 10001 --gid edupilot edupilot
WORKDIR /srv/edupilot
ENV PATH="/srv/edupilot:$PATH"
COPY --from=go-build /out/ /srv/edupilot/
COPY --from=go-build /src/.env.example .env
COPY --from=web-build /src/clients/web/dist ./clients/web/dist
RUN mkdir -p database storage && chown -R edupilot:edupilot database storage
USER edupilot
EXPOSE 3000
CMD ["api"]

FROM nginx:stable-alpine AS web
COPY docker/web.conf /etc/nginx/conf.d/default.conf
COPY --from=web-build /src/clients/web/dist/ /usr/share/nginx/html/
EXPOSE 8080
