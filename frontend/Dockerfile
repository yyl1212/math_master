FROM node:24.17.0-alpine@sha256:9e04e3f9c9164cb2c913593ba0733fbd0caf4e9bea474d743ca8a32aa36d98c9 AS build
WORKDIR /src/frontend
ENV NEXT_TELEMETRY_DISABLED=1 NODE_OPTIONS=--max-old-space-size=1536
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
COPY content/catalogue/ /src/content/catalogue/
COPY tests/e2e/ /src/tests/e2e/
RUN mkdir -p public && npm run build

FROM node:24.17.0-alpine@sha256:9e04e3f9c9164cb2c913593ba0733fbd0caf4e9bea474d743ca8a32aa36d98c9
ARG VCS_REF
LABEL org.opencontainers.image.revision=$VCS_REF
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 HOSTNAME=0.0.0.0 PORT=3000
COPY --from=build --chown=node:node /src/frontend/.next/standalone/ ./
COPY --from=build --chown=node:node /src/frontend/.next/static/ ./.next/static/
COPY --from=build --chown=node:node /src/frontend/public/ ./public/
USER node
ENTRYPOINT ["node", "server.js"]
