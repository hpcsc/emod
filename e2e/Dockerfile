FROM golang:1.25-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG BUILD_TAG=v0.1.0
RUN CGO_ENABLED=0 go build -ldflags "-X github.com/hpcsc/emod/internal/version.releaseTag=${BUILD_TAG}" -o ./dist/emod ./cmd/emod

FROM node:24-slim
WORKDIR /app

RUN apt-get update && apt-get install -y \
    python3 \
    make \
    g++ \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/dist/emod /app/dist/emod
COPY --from=builder /app/internal/parser/testdata /app/internal/parser/testdata
COPY e2e /app/e2e
ENTRYPOINT ["/app/e2e/entrypoint.sh"]
