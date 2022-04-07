FROM golang:1.18.0-alpine3.15

RUN apk add bash git --no-cache
WORKDIR /go/src/chain

RUN PATH="$(go env GOPATH)/bin:$CHAIN/bin:$PATH" && \
    go env -w GO111MODULE=off

EXPOSE 1999