FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/plane-tools ./cmd/plane-tools

FROM alpine:3.22
RUN addgroup -S -g 10001 plane-tools && adduser -S -D -H -u 10001 -G plane-tools plane-tools
COPY --from=build /out/plane-tools /usr/local/bin/plane-tools
RUN mkdir -p /data && chown plane-tools:plane-tools /data
USER plane-tools
EXPOSE 8080
VOLUME ["/data"]
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD wget -q -O - http://127.0.0.1:8080/healthz >/dev/null || exit 1
ENTRYPOINT ["/usr/local/bin/plane-tools"]
