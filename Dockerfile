# syntax=docker/dockerfile:1

# --- Build stage ---
FROM golang:1.26 AS build

WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

# Build a static binary (no libc, works on distroless/static).
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /sweds .

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /sweds /sweds
# Baked sample config; override with a bind-mount at /config.yaml.
COPY config.yaml /config.yaml

# Writable home for the nonroot user; the default "./data" dir lands here.
WORKDIR /home/nonroot

ENV CONFIG_PATH=/config.yaml
EXPOSE 8080

USER nonroot
ENTRYPOINT ["/sweds"]
