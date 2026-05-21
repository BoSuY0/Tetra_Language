FROM golang:1.20-bookworm AS build

WORKDIR /src/compiler
COPY compiler/go.mod ./
COPY compiler ./
RUN go build -trimpath -ldflags="-s -w" -o /out/tetra-techempower ./cmd/tetra-techempower

FROM debian:bookworm-slim

RUN apt-get update \
  && apt-get install -y --no-install-recommends curl ca-certificates \
  && rm -rf /var/lib/apt/lists/*
RUN useradd --system --home /nonexistent --shell /usr/sbin/nologin tetra
COPY --from=build /out/tetra-techempower /usr/local/bin/tetra-techempower

ENV TETRA_TE_HOST=0.0.0.0
ENV TETRA_TE_PORT=8080
ENV TETRA_TE_WORKERS=
ENV TETRA_TE_PG_HOST=tfb-database
ENV TETRA_TE_PG_PORT=5432
ENV TETRA_TE_PG_USER=benchmarkdbuser
ENV TETRA_TE_PG_DATABASE=hello_world
ENV TETRA_TE_PG_PASSWORD=benchmarkdbpass
ENV TETRA_TE_PG_POOL=256

EXPOSE 8080
HEALTHCHECK --interval=2s --timeout=2s --retries=30 CMD curl -fsS http://127.0.0.1:8080/plaintext >/dev/null || exit 1
USER tetra
CMD ["/usr/local/bin/tetra-techempower"]
