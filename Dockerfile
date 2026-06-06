# syntax=docker/dockerfile:1

FROM golang:1.20-bookworm AS build

WORKDIR /src
COPY . .

ARG TETRA_VERSION=v0.4.0

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/tetra ./cli/cmd/tetra \
    && cp /out/tetra /out/t

FROM debian:bookworm-slim

ARG TETRA_VERSION=v0.4.0

LABEL org.opencontainers.image.title="Tetra Language"
LABEL org.opencontainers.image.description="Tetra Language CLI and compiler"
LABEL org.opencontainers.image.source="https://github.com/BoSuY0/Tetra_Language"
LABEL org.opencontainers.image.version="${TETRA_VERSION}"
LABEL org.opencontainers.image.licenses="Apache-2.0"

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/tetra /usr/local/bin/tetra
COPY --from=build /out/t /usr/local/bin/t

WORKDIR /work
CMD ["tetra", "version"]
