FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine3.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG TARGETOS TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/lyra ./cmd/lyra

FROM alpine:3.24
RUN apk add --no-cache git git-daemon ca-certificates \
    && adduser -D -H -u 1000 lyra \
    && mkdir /data \
    && chown lyra:lyra /data
COPY --from=build /out/lyra /usr/local/bin/lyra
USER lyra
ENV LYRA_LISTEN=:8080 LYRA_DATA=/data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["lyra"]
CMD ["serve"]
