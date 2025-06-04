FROM golang:1.24-bookworm as BUILD
WORKDIR /app/

COPY server/go.mod server/go.sum ./
RUN go mod download
RUN go mod tidy

COPY server/ ./

RUN go build -o ./build/server ./cmd

RUN mkdir lib
RUN cp "$(ldd ./build/server | awk '/libgcc_s.so.1/ {print $3}')" lib/libgcc_s.so.1 || :
RUN cp /lib/x86_64-linux-gnu/libgcc_s.so.1 lib/libgcc_s.so.1 || :

FROM gcr.io/distroless/base-debian12:debug
WORKDIR /usr/local/bin
COPY --from=BUILD /app/build/server /usr/local/bin/server
COPY --from=BUILD /app/lib /usr/lib
EXPOSE 50051
ENTRYPOINT ["/usr/local/bin/server"] 