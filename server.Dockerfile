FROM golang:1.24-bookworm AS build
WORKDIR /app/

COPY . .

RUN go mod download
RUN go mod tidy

RUN go build -o ./build/server ./server/cmd

RUN mkdir -p lib
RUN cp "$(ldd ./build/signer_server | awk '/libgcc_s.so.1/ {print $3}')" lib/libgcc_s.so.1 || :
RUN cp /lib/x86_64-linux-gnu/libgcc_s.so.1 lib/libgcc_s.so.1 || :

FROM debian:bookworm-slim
WORKDIR /usr/local/bin

RUN apt-get update && \
    apt-get install -y ca-certificates sqlite3 && \
    update-ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=build /app/build/server /usr/local/bin/server
COPY --from=build /app/lib /usr/lib
COPY --from=build /app/migrations /usr/local/bin/migrations

EXPOSE 9006 9007
ENTRYPOINT ["server", "-config"]
