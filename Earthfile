VERSION 0.6
FROM golang:1.19-alpine3.16
WORKDIR "/code"
RUN apk add alpine-sdk ca-certificates
ARG BUILD_FLAGS='-tags musl'

tidy:
    LOCALLY
    RUN go mod tidy

fmt:
    LOCALLY
    RUN go fmt ./...

build:
    FROM +sources
    RUN make BINARY=mqtt-proxy BUILD_FLAGS="${BUILD_FLAGS}" GOOS=linux GOARCH=amd64 build

test:
   FROM +sources
   RUN go test -mod=vendor ${BUILD_FLAGS} -v ./...

sources:
    COPY . /code
