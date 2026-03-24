FROM node:24-alpine AS frontend-builder

WORKDIR /build

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# ---

FROM golang:1.24-alpine AS go-builder

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /build/dist ./frontend/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="-s -w -X 'github.com/swopcart/server/internal/config.Version=${VERSION}'" \
    -o swopcartd ./cmd/swopcartd

# ---

FROM alpine:3

RUN apk add --no-cache su-exec tzdata

COPY --from=go-builder /build/swopcartd /usr/local/bin/swopcartd
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod +x /usr/local/bin/docker-entrypoint.sh

ENV SWOPCART_DATA=/data
VOLUME /data
EXPOSE 8000

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["/usr/local/bin/swopcartd"]
