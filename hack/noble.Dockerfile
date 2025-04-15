# Info on how to use this docker image can be found in DOCKER_README.md
ARG IMG_TAG=latest

# Compile the nobled binary
FROM --platform=linux/amd64 golang:1.23-alpine AS noble-builder
WORKDIR /src/
ENV PACKAGES="curl make git libc-dev bash file gcc linux-headers eudev-dev"
RUN apk add --no-cache $PACKAGES

ARG CHAIN_TAG=main
RUN git clone --depth 1 --branch $CHAIN_TAG https://github.com/noble-assets/noble /src/app
WORKDIR /src/app

COPY replaces.sh .
RUN chmod +x replaces.sh && sh replaces.sh
RUN cat go.mod
RUN go mod tidy

RUN LEDGER_ENABLED=false BUILD_TAGS=muslc make build

FROM --platform=linux/amd64 alpine:$IMG_TAG
RUN apk add --no-cache build-base jq
RUN addgroup -g 1025 nonroot
RUN adduser -D nonroot -u 1025 -G nonroot
ARG IMG_TAG
COPY --from=noble-builder  /src/app/build /usr/local/bin/noble
EXPOSE 26656 26657 1317 9090
USER nonroot

ENTRYPOINT ["noble", "start"]