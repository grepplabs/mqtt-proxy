FROM golang:1.20-alpine3.17 as builder

RUN apk add alpine-sdk ca-certificates

ARG TARGETOS
ARG TARGETARCH

WORKDIR "/code"
ADD . "/code"

# https://github.com/confluentinc/confluent-kafka-go: When building your application for Alpine Linux (musl libc) you must pass -tags musl to go get, go build, etc.
RUN make BINARY=mqtt-proxy BUILD_FLAGS="-tags musl" GOOS=${TARGETOS} GOARCH=${TARGETARCH} build

FROM alpine:3.17
RUN apk add ca-certificates
COPY --from=builder /code/mqtt-proxy /mqtt-proxy
ENTRYPOINT ["/mqtt-proxy"]
