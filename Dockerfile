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

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./apps/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/worker ./apps/worker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/migrate ./apps/migrate

FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
  bash \
  ca-certificates \
  fonts-liberation \
  libasound2 \
  libatk-bridge2.0-0 \
  libatk1.0-0 \
  libc6 \
  libcairo2 \
  libcups2 \
  libdbus-1-3 \
  libexpat1 \
  libfontconfig1 \
  libgbm1 \
  libgcc-s1 \
  libglib2.0-0 \
  libgtk-3-0 \
  libnspr4 \
  libnss3 \
  libpango-1.0-0 \
  libpangocairo-1.0-0 \
  libstdc++6 \
  libx11-6 \
  libx11-xcb1 \
  libxcb1 \
  libxcomposite1 \
  libxcursor1 \
  libxdamage1 \
  libxext6 \
  libxfixes3 \
  libxi6 \
  libxrandr2 \
  libxrender1 \
  libxshmfence1 \
  libxss1 \
  libxtst6 \
  postgresql-client \
  python3 \
  python3-venv \
  python3-pip \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=web-builder /src/web/dist /app/web-dist
COPY --from=backend-builder /out/api /app/api
COPY --from=backend-builder /out/worker /app/worker
COPY --from=backend-builder /out/migrate /app/migrate
COPY scripts/python /app/scripts/python
COPY docker/start-all-in-one.sh /app/start.sh

RUN chmod +x /app/api /app/worker /app/migrate /app/start.sh

EXPOSE 8080

ENTRYPOINT ["/app/start.sh"]
