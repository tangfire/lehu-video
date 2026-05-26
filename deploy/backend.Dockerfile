FROM golang:1.25-bookworm AS builder

WORKDIR /src
ENV CGO_ENABLED=0
ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

ARG SERVICE_PATH
RUN go build -ldflags "-X main.Version=docker" -o /out/service ${SERVICE_PATH}

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/service /app/service

WORKDIR /app

EXPOSE 8080 8020 8030 8040 9020 9030 9040

ENTRYPOINT ["/app/service"]
