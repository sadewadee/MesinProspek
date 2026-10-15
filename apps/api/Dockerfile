# Image of the single binary `msp` (api, worker, migrate, tool): <registry>/msp/api:<git-sha> (docs/06 §4, §5).
# Build context = apps/api, as in the build job of .gitlab-ci.yml:
#   docker build -f apps/api/Dockerfile -t msp/api apps/api
# The binary is static (CGO off), so it runs on Alpine, which has wget for the Compose healthcheck (06 §5).

# Same Go image as the CI check job (.gitlab-ci.yml).
FROM golang:1.26.9-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/msp ./cmd/msp

FROM alpine:3.24.2
# Non-root user. MSP_STORAGE_DIR exists and belongs to it, so a fresh named volume
# mounted there (docs/04 §13) inherits the ownership.
RUN addgroup -S -g 10001 msp \
 && adduser -S -D -H -u 10001 -G msp msp \
 && mkdir -p /var/lib/msp/storage \
 && chown msp:msp /var/lib/msp/storage
COPY --from=build /out/msp /usr/local/bin/msp
USER msp
# :8080 HTTP (msp api), :9090 metrics (internal listener, never routed by Caddy)
EXPOSE 8080 9090
CMD ["msp", "api"]
