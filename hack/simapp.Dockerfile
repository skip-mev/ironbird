ARG IMG_TAG=latest

# Compile the simapp binary
FROM --platform=linux/amd64 golang:1.23-alpine AS simd-builder
WORKDIR /src/
ARG GIT_SHA
RUN echo "Ironbird building with SHA: $GIT_SHA"

ENV PACKAGES="curl make git libc-dev bash file gcc linux-headers eudev-dev"
RUN apk add --no-cache $PACKAGES

ARG CHAIN_TAG=main
ARG REPLACE_CMD
RUN git clone --depth 1 --branch $CHAIN_TAG https://github.com/cosmos/cosmos-sdk /src/app

WORKDIR /src/app/simapp
RUN echo "$REPLACE_CMD" > replace_cmd.sh
RUN chmod +x replace_cmd.sh && sh replace_cmd.sh
RUN cat go.mod
RUN go mod tidy
WORKDIR /src/app

RUN make build

FROM --platform=linux/amd64 alpine:$IMG_TAG
RUN apk add --no-cache build-base jq
RUN addgroup -g 1025 nonroot
RUN adduser -D nonroot -u 1025 -G nonroot
ARG IMG_TAG
COPY --from=simd-builder  /src/app/build/simd /usr/bin/simd
EXPOSE 26656 26657 1317 9090 26660
USER nonroot

ENTRYPOINT ["simd", "start"]