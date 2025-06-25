# Info on how to use this docker image can be found in DOCKER_README.md
ARG IMG_TAG=latest

# Compile the gaiad binary
FROM golang:1.23-alpine AS gaiad-builder
ARG GIT_SHA
RUN echo "Ironbird building with SHA: $GIT_SHA"
WORKDIR /src/
ENV PACKAGES="curl make git libc-dev bash file gcc g++ gcc-gnat linux-headers eudev-dev libstdc++"
RUN apk add --no-cache $PACKAGES

# See https://github.com/CosmWasm/wasmvm/releases
ARG WASMVM_VERSION=v2.2.4
ADD https://github.com/CosmWasm/wasmvm/releases/download/${WASMVM_VERSION}/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
ADD https://github.com/CosmWasm/wasmvm/releases/download/${WASMVM_VERSION}/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
RUN cp "/lib/libwasmvm_muslc.$(uname -m).a" /lib/libwasmvm_muslc.a

ARG CHAIN_TAG
ARG CHAIN_SRC=https://github.com/cosmos/gaia
ARG REPLACE_CMD

RUN git clone $CHAIN_SRC /src/app && \
    cd /src/app && \
    git checkout $CHAIN_TAG

WORKDIR /src/app
RUN echo "$REPLACE_CMD" > replace_cmd.sh
RUN chmod +x replace_cmd.sh && sh replace_cmd.sh
RUN cat go.mod
RUN go mod tidy
    
COPY . .

ENV CGO_ENABLED=1
ENV CGO_LDFLAGS="-L/lib -lwasmvm_muslc"
RUN LEDGER_ENABLED=false LINK_STATICALLY=true BUILD_TAGS="muslc netgo" make build
RUN echo "Ensuring binary is statically linked ..."  \
    && file /src/app/build/gaiad | grep "statically linked"

FROM alpine:$IMG_TAG
RUN apk add --no-cache build-base jq
RUN addgroup -g 1025 nonroot
RUN adduser -D nonroot -u 1025 -G nonroot
ARG IMG_TAG
COPY --from=gaiad-builder  /src/app/build/gaiad /usr/local/bin/
EXPOSE 26656 26657 1317 9090 26660
USER nonroot

ENTRYPOINT ["gaiad", "start"]
