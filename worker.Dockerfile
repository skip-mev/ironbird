FROM golang:1.25-bookworm AS builder
WORKDIR /app

RUN mkdir -p /root/.cache/go-build
RUN go env -w GOMODCACHE=/root/.cache/go-build

COPY go.mod go.sum Makefile ./
RUN --mount=type=cache,target=/root/.cache/go-build make deps

COPY . .

RUN make build

FROM alpine:latest

RUN apk add --no-cache ca-certificates libc6-compat gcompat

COPY --from=builder /app/build/worker /usr/local/bin/worker

ENTRYPOINT ["worker", "-config"]
