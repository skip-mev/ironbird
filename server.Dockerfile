FROM golang:1.24-bookworm as build
WORKDIR /app/

COPY . ./

RUN go mod download
RUN go mod tidy

RUN go build -o ./build/server ./server/cmd

FROM alpine:latest
WORKDIR /usr/local/bin
COPY --from=build /app/build/server /usr/local/bin/server
EXPOSE 50051
ENTRYPOINT ["/usr/local/bin/server"] 