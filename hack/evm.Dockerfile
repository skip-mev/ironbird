ARG IMG_TAG=latest

# Compile the simapp binary
FROM golang:1.25-alpine AS evmd-builder
ARG GIT_SHA
RUN echo "Ironbird building with SHA: $GIT_SHA"

WORKDIR /src/

ENV PACKAGES="curl make git libc-dev bash file build-base linux-headers eudev-dev"
RUN apk add --no-cache $PACKAGES

ARG CHAIN_TAG
ARG CHAIN_SRC=https://github.com/cosmos/evm
ARG REPLACE_CMD

RUN git clone $CHAIN_SRC /src/app && \
  cd /src/app && \
  git checkout $CHAIN_TAG

WORKDIR /src/app/evmd
RUN if [ -n "$REPLACE_CMD" ]; then \
        go mod tidy && \
        echo "After go mod tidy, applying replace commands:" && \
        echo "$REPLACE_CMD" > replace_cmd.sh && \
        chmod +x replace_cmd.sh && \
        sh replace_cmd.sh && \
        echo "Final go.mod:" && \
        cat go.mod && \
        echo "Updating go.sum with replaced modules:" && \
        go get ./... && \
        echo "Done updating go.sum"; \
    else \
        go mod tidy; \
    fi
WORKDIR /src/app

RUN make build

FROM alpine:$IMG_TAG
RUN apk add --no-cache build-base jq curl
RUN apk add --no-cache tini
RUN addgroup -g 1025 nonroot
RUN adduser -D nonroot -u 1025 -G nonroot
RUN mkdir -p /home/nonroot && chown nonroot:nonroot /home/nonroot
ARG IMG_TAG
COPY evmd-entrypoint.sh /usr/bin/entrypoint.sh
RUN chmod +x /usr/bin/entrypoint.sh
COPY --from=evmd-builder  /src/app/build/evmd /usr/bin/evmd
EXPOSE 26656 26657 1317 9090 26660 8545 8100 9464
USER nonroot
WORKDIR /home/nonroot
RUN test -x /sbin/tini && test -x /usr/bin/entrypoint.sh

ENTRYPOINT ["/sbin/tini", "--", "/usr/bin/entrypoint.sh"]
