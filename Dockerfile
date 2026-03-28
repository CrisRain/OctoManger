FROM oven/bun:1 AS web-builder
WORKDIR /src/web

COPY apps/web/package.json ./
COPY apps/web/bun.lock* ./
RUN if [ -f bun.lock ] || [ -f bun.lockb ]; then bun install --frozen-lockfile; else bun install; fi

COPY apps/web/. .
COPY contracts/ ../contracts/
RUN bun run build

FROM golang:1.26-alpine AS backend-builder
WORKDIR /src
ARG TARGETOS=linux
ARG TARGETARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=web-builder /src/web/dist /src/internal/platform/webui/dist
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/octomanger ./apps/octomanger

FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
  bash \
  ca-certificates \
  curl \
  postgresql-client \
  python3 \
  python3-venv \
  python3-pip \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=backend-builder /out/octomanger /app/octomanger
COPY scripts/python /app/scripts/python

RUN chmod +x /app/octomanger

EXPOSE 8080

ENTRYPOINT ["/app/octomanger"]
