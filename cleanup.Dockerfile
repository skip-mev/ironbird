FROM golang:1.25-bookworm AS builder
WORKDIR /app

RUN mkdir -p /root/.cache/go-build
RUN go env -w GOMODCACHE=/root/.cache/go-build

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build go mod download

COPY . .

RUN go build -o ./build/cleanup ./cmd/cleanup

FROM alpine:latest

RUN apk add --no-cache ca-certificates libc6-compat gcompat

COPY --from=builder /app/build/cleanup /app/cleanup

ENTRYPOINT ["/app/cleanup"]
