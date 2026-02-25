# ---- Build stage ----
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates

WORKDIR /src
COPY . .

# Static binary, stripped of debug info
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -trimpath \
    -o /proxy .

# ---- Runtime stage ----
FROM scratch

# TLS root certs
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Binary
COPY --from=builder /proxy /proxy

EXPOSE 80

# The app requires a config file as the first argument.
# It should be mounted at /config.yaml at runtime.
ENTRYPOINT ["/proxy", "/config.yaml"]
