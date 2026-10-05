FROM golang:1.27.1-alpine@sha256:cd9a32216aee5667f957a62d13a10032a63fd58e14b3f3d9cc8c2122f501e95e AS build
ENV CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=amd64
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
COPY schemas/go.mod /src/schemas/go.mod
RUN go mod download
COPY schemas/ /src/schemas/
COPY backend/ /src/backend/
RUN mkdir /out && for command in server migrate admin-init correction-maintenance; do go build -trimpath -ldflags="-s -w" -o /out/$command ./cmd/$command; done

FROM alpine:3.24@sha256:d56c381f961d307a21b3ca004cf1e3910f106644aefb1f43e654c8a56c4fd395
ARG VCS_REF
LABEL org.opencontainers.image.revision=$VCS_REF
RUN apk add --no-cache ca-certificates && addgroup -S -g 10001 app && adduser -S -D -H -u 10001 -G app app
WORKDIR /app
COPY --from=build --chown=10001:10001 /out/ /app/bin/
COPY --chown=10001:10001 db/migrations/ /app/db/migrations/
USER 10001:10001
ENTRYPOINT ["/app/bin/server"]
