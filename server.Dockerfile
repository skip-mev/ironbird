FROM golang:1.24-bookworm AS build
WORKDIR /app/

COPY . .

RUN go mod download
RUN go mod tidy

RUN go build -o ./build/server ./server/cmd

RUN cp "$(ldd ./build/signer_server | awk '/libgcc_s.so.1/ {print $3}')" lib/libgcc_s.so.1 || :
RUN cp /lib/x86_64-linux-gnu/libgcc_s.so.1 lib/libgcc_s.so.1 || :

RUN apt-get update && apt-get install -y sqlite3

FROM gcr.io/distroless/base-debian12:debug
WORKDIR /usr/local/bin
COPY --from=build /app/build/server /usr/local/bin/server
COPY --from=build /app/lib /usr/lib
COPY --from=build /app/migrations /usr/local/bin/migrations
COPY --from=build /usr/bin/sqlite3 /usr/local/bin/sqlite3
COPY --from=build /usr/lib/x86_64-linux-gnu/libsqlite3.so.0 /usr/lib/x86_64-linux-gnu/libsqlite3.so.0
COPY --from=build /lib/x86_64-linux-gnu/libreadline.so.8 /lib/x86_64-linux-gnu/libreadline.so.8
COPY --from=build /lib/x86_64-linux-gnu/libncurses.so.6 /lib/x86_64-linux-gnu/libncurses.so.6
EXPOSE 9006 9007
ENTRYPOINT ["server", "-config"]