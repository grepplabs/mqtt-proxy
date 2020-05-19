FROM golang:1.14-alpine3.11 as builder

RUN apk add alpine-sdk

WORKDIR "/code"
ADD . "/code"

# https://github.com/confluentinc/confluent-kafka-go: When building your application for Alpine Linux (musl libc) you must pass -tags musl to go get, go build, etc.
RUN make BINARY=mqtt-proxy BUILD_FLAGS="-tags musl" GOOS=linux GOARCH=amd64 build

FROM alpine:3.11
COPY --from=builder /code/mqtt-proxy /mqtt-proxy
ENTRYPOINT ["/mqtt-proxy"]
