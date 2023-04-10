FROM golang:1.20-alpine3.16 as builder

RUN apk add alpine-sdk ca-certificates

WORKDIR "/code"
ADD . "/code"

# https://github.com/confluentinc/confluent-kafka-go: When building your application for Alpine Linux (musl libc) you must pass -tags musl to go get, go build, etc.
RUN make BINARY=mqtt-proxy BUILD_FLAGS="-tags musl" GOOS=linux GOARCH=amd64 build

FROM alpine:3.16
RUN apk add ca-certificates
COPY --from=builder /code/mqtt-proxy /mqtt-proxy
ENTRYPOINT ["/mqtt-proxy"]
