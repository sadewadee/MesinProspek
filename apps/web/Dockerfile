# Image of the Next.js app: <registry>/msp/web:<git-sha> (docs/06 §4, §5), `output: "standalone"`.
# Build context = repo root, because apps/web uses pnpm workspace packages (packages/*); the CI build
# job does the same:
#   docker build -f apps/web/Dockerfile -t msp/web .
# Node version = .nvmrc.
# Build once for every environment: no NEXT_PUBLIC_* values (docs/06 §4). Runtime settings come
# from the environment of the container (MSP_KRATOS_PUBLIC_URL, read on the server only).

FROM node:22.23.3-alpine3.24 AS build
RUN corepack enable
WORKDIR /repo
# Manifests first, so the dependency layer is cached until a manifest or the lockfile changes.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY apps/web/package.json apps/web/
COPY packages/api-client/package.json packages/api-client/
COPY packages/design-tokens/package.json packages/design-tokens/
COPY packages/ui/package.json packages/ui/
RUN pnpm install --frozen-lockfile --filter @msp/web...
COPY apps/web/ apps/web/
COPY packages/ packages/
ENV NEXT_TELEMETRY_DISABLED=1
RUN pnpm --filter @msp/web build

FROM node:22.23.3-alpine3.24
# HOSTNAME: Docker sets it to the container ID and the Next.js server binds to it; 0.0.0.0 also
# serves 127.0.0.1, which the Compose healthcheck calls.
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=3000 HOSTNAME=0.0.0.0
WORKDIR /app
# The standalone tree mirrors the workspace (server at apps/web/server.js); static assets are not part of it.
COPY --from=build --chown=node:node /repo/apps/web/.next/standalone ./
COPY --from=build --chown=node:node /repo/apps/web/.next/static ./apps/web/.next/static
USER node
EXPOSE 3000
CMD ["node", "apps/web/server.js"]
