FROM golang:1.24-alpine3.22 AS build
WORKDIR /app/

COPY go.mod go.sum .

RUN go mod download
RUN go mod tidy

COPY . .

ENV CGO_ENABLED=1
RUN apk add --no-cache gcc musl-dev
RUN go build -ldflags '-s -w -extldflags "--static"' -o ./build/server ./server/cmd/

RUN mkdir lib
RUN cp "$(ldd ./build/signer_server | awk '/libgcc_s.so.1/ {print $3}')" lib/libgcc_s.so.1 || :
RUN cp /lib/x86_64-linux-gnu/libgcc_s.so.1 lib/libgcc_s.so.1 || :

FROM gcr.io/distroless/base-debian12:debug
WORKDIR /usr/local/bin
COPY --from=BUILD /app/build/server /usr/local/bin/server
COPY --from=BUILD /app/lib /usr/lib
COPY --from=BUILD /app/migrations /usr/local/bin/migrations
EXPOSE 9006 9007
ENTRYPOINT ["server", "-config"]
