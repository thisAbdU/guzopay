# Debian-based builder (GCC preinstalled; avoids slow Alpine apk during CGO build)
FROM golang:1.22-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /guzopay .

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /guzopay /usr/local/bin/guzopay

WORKDIR /app

EXPOSE 8080

CMD ["guzopay"]
