# mqtt-proxy

**Work in progress**

![Build](https://github.com/grepplabs/mqtt-proxy/workflows/build/badge.svg)
[![Docker Hub](https://img.shields.io/badge/docker-latest-blue.svg)](https://hub.docker.com/r/grepplabs/mqtt-proxy)
[![Docker Pulls](https://img.shields.io/docker/pulls/grepplabs/mqtt-proxy)](https://hub.docker.com/r/grepplabs/mqtt-proxy)

MQTT Proxy allows MQTT clients to send messages to other messaging systems


## Implementation status

* MQTT protocol
    * [x] [MQTT 3.1.1](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/mqtt-v3.1.1.html)
    * [ ] [MQTT 5.0](https://docs.oasis-open.org/mqtt/mqtt/v5.0/mqtt-v5.0.html)
* Publisher
    * [x] [Apache Kafka](https://kafka.apache.org/)
    * [ ] [Apache Pulsar](https://pulsar.apache.org/)
    * [ ] Others
* [ ] Authentication
* [ ] Helm chart

## Build
### build binary

    make clean build

### build docker image

    make clean docker-build

## Test

prerequisites
- [docker compose](https://docs.docker.com/compose/install/)

### kafka publisher

1. build and start-up test environment

    ```
    cd scripts/cp-kafka
    make build-up
    ```

2. subscribe to Kafka topic

    ```
    docker exec -it broker kafka-console-consumer --bootstrap-server localhost:9092 --topic mqtt-test --property print.key=true --from-beginning
    ```

3. publish messages using mosquitto client

    ```
    docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 0" --repeat 1 -q 0
    docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 1" --repeat 1 -q 1
    docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 2" --repeat 1 -q 2
    ```

4. check the prometheus metrics

    ```
    watch -c 'curl -s localhost:9090/metrics | grep mqtt | egrep -v '^#''
    ```

## Metrics


metric | labels | description
-------| -------| ------------
|mqtt_proxy_build_info| branch, goversion, revision, revision|A metric with a constant '1' value labeled by version, revision, branch, and goversion from which mqtt_proxy was built.|
|mqtt_proxy_server_connections_active| |Number of active TCP connections from clients to server.|
|mqtt_proxy_server_connections_total| |Total number of TCP connections from clients to server.|
|mqtt_proxy_handler_requests_total|type|Total number of MQTT requests labeled by package control type. |
|mqtt_proxy_handler_responses_total|type|Total number of MQTT responses labeled by package control type. |
|mqtt_proxy_publisher_publish_duration_seconds | name, type, qos | Histogram tracking latencies for publish requests. |