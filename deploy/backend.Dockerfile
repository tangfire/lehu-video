FROM golang:1.25-bookworm AS builder

WORKDIR /src
ENV CGO_ENABLED=0
ENV GOPROXY=https://goproxy.cn,direct

RUN apt-get update && apt-get install -y --no-install-recommends \
    fonts-noto-cjk \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

ARG SERVICE_PATH
RUN go build -ldflags "-X main.Version=docker" -o /out/service ${SERVICE_PATH}
RUN go build -o /out/healthcheck ./cmd/healthcheck

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/fonts /usr/share/fonts
COPY --from=builder /out/service /app/service
COPY --from=builder /out/healthcheck /app/healthcheck

WORKDIR /app

EXPOSE 8080 8020 8030 8040 9020 9030 9040

ENTRYPOINT ["/app/service"]
