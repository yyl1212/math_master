FROM alpine:3.24@sha256:d56c381f961d307a21b3ca004cf1e3910f106644aefb1f43e654c8a56c4fd395 AS download
RUN apk add --no-cache curl ca-certificates
RUN curl -fsSL --retry 3 --connect-timeout 10 --max-time 120 https://github.com/caddyserver/caddy/releases/download/v2.11.7/caddy_2.11.7_linux_amd64.tar.gz -o /tmp/caddy.tar.gz \
    && echo '727b91701a392de6ebc5027509f548bf39979e5216340d0faed8fa5e69c84f8b  /tmp/caddy.tar.gz' | sha256sum -c - \
    && mkdir /out && tar -xzf /tmp/caddy.tar.gz -C /out caddy

FROM alpine:3.24@sha256:d56c381f961d307a21b3ca004cf1e3910f106644aefb1f43e654c8a56c4fd395
ARG VCS_REF
LABEL org.opencontainers.image.revision=$VCS_REF
ENV XDG_CONFIG_HOME=/config XDG_DATA_HOME=/data
RUN apk add --no-cache ca-certificates && addgroup -S -g 10001 app && adduser -S -D -H -u 10001 -G app app \
    && mkdir -p /data /config /etc/caddy && chown -R 10001:10001 /data /config /etc/caddy
COPY --from=download /out/caddy /usr/bin/caddy
USER 10001:10001
ENTRYPOINT ["/usr/bin/caddy"]
CMD ["run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
