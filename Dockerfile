FROM golang:1.25-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git make build-base

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags '-extldflags "-static"' \
    -o macnet .

FROM alpine:latest

RUN apk --no-cache add ca-certificates git

WORKDIR /app

COPY --from=builder /build/macnet /usr/local/bin/macnet

RUN addgroup -S macnet && adduser -S macnet -G macnet
USER macnet

ENTRYPOINT ["macnet"]
CMD ["--help"]
