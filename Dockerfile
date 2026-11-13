FROM --platform=$BUILDPLATFORM alpine:3.24 AS binary
ARG TARGETOS TARGETARCH
COPY dist/lyra-*-${TARGETOS}-${TARGETARCH}.tar.gz /tmp/lyra.tar.gz
RUN tar -xzf /tmp/lyra.tar.gz -C /usr/local/bin lyra

FROM alpine:3.24
RUN apk add --no-cache git git-daemon ca-certificates \
    && adduser -D -H -u 1000 lyra \
    && mkdir /data \
    && chown lyra:lyra /data
COPY --from=binary /usr/local/bin/lyra /usr/local/bin/lyra
USER lyra
ENV LYRA_LISTEN=:8080 LYRA_DATA=/data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["lyra"]
CMD ["serve"]
