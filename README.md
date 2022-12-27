# mqtt-proxy

**Work in progress**

[![License: CC BY-NC-ND 4.0](https://img.shields.io/badge/License-CC%20BY--NC--ND%204.0-lightgrey.svg)](https://creativecommons.org/licenses/by-nc-nd/4.0/)
![Build](https://github.com/grepplabs/mqtt-proxy/workflows/build/badge.svg)
[![Docker Hub](https://img.shields.io/badge/docker-latest-blue.svg)](https://hub.docker.com/r/grepplabs/mqtt-proxy)
[![Docker Pulls](https://img.shields.io/docker/pulls/grepplabs/mqtt-proxy)](https://hub.docker.com/r/grepplabs/mqtt-proxy)

MQTT Proxy allows MQTT clients to send messages to other messaging systems


## Implementation status

* MQTT protocol
    * [x] [MQTT 3.1.1](http://docs.oasis-open.org/mqtt/mqtt/v3.1.1/mqtt-v3.1.1.html)
    * [ ] [MQTT 5.0](https://docs.oasis-open.org/mqtt/mqtt/v5.0/mqtt-v5.0.html)
* Publisher
    * [x] Noop
    * [x] [Apache Kafka](https://kafka.apache.org/)
    * [ ] Others
* Authentication
    * [x] Noop
    * [x] Plain
    * [ ] Others
* [x] Helm chart
* [x] Client certificate revocation list
* [ ] Server certificates rotation
    * [x] Files certificate source
    * [ ] HashiCorp Vault certificate source


### Install binary release

1. Download the latest Linux release

    ```
    TAG=$(curl --silent "https://api.github.com/repos/grepplabs/mqtt-proxy/releases/latest" | jq -r '.tag_name')
    curl -Ls https://github.com/grepplabs/mqtt-proxy/releases/download/${TAG}/mqtt-proxy-${TAG}-linux-amd64.tar.gz | tar xz
    ```

2. Move the binary in to your PATH.

    ```
    sudo mv ./mqtt-proxy /usr/local/bin/mqtt-proxy
    ```

## Build
### build binary

    make clean build

### build docker image

    make clean docker-build

## Helm 3 chart

Deploy the Helm chart

```bash
git clone git@github.com:grepplabs/mqtt-proxy.git
helm install mqtt-proxy ./mqtt-proxy/charts/mqtt-proxy \
  --set image.tag=latest \
  --set image.repository=grepplabs/mqtt-proxy \
  --values  <(echo '{
        "extraArgs" : ["server","--mqtt.publisher.name=noop"]
    }')
```

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

    * proxy using Kafka PLAINTEXT listener
    ```
    docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 0" --repeat 1 -q 0
    docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 1" --repeat 1 -q 1
    docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 2" --repeat 1 -q 2
    ```

    * proxy using Kafka SSL listener
    ```
    docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy-ssl:1884/dummy -m "test qos 0" --repeat 1 -q 0
    docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy-ssl:1884/dummy -m "test qos 1" --repeat 1 -q 1
    docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy-ssl:1884/dummy -m "test qos 2" --repeat 1 -q 2
    ```

4. check the prometheus metrics

    ```
    watch -c 'curl -s localhost:9090/metrics | grep mqtt | egrep -v '^#''
    ```

5. see also [cp-kafka](scripts/cp-kafka/Makefile) with SASL_PLAINTEXT and SASL_SSL configuration

### publish to Amazon MSK

1. provision test MSK and EC2 running in [podman](https://podman.io/) 2 proxy containers

    ```
    cd scripts/msk
    make tf-apply
    ```

2. create Kafka mqtt-test topic

3. publish

    * container connects to MSK PLAINTEXT listener
    ```
    mosquitto_pub -m "on" -t "dummy" -k 20 -i mqtt-proxy.clientv --repeat 1 -q 1 -h <ec2-ip> -p 1883
    ```

    * container connects to MSK TLS listener
    ```
    mosquitto_pub -m "on" -t "dummy" -k 20 -i mqtt-proxy.clientv --repeat 1 -q 1 -h <ec2-ip> -p 1884
    ```

### plain authenticator

1. start server with `plain` authenticator
    * with credentials file

    ```
    cat <<EOF > mqtt-credentials.csv
    alice,alice-secret
    "bob","bob-secret"
    EOF
    ```

    ```
    mqtt-proxy server --mqtt.publisher.name=noop \
        --mqtt.handler.auth.name=plain \
        --mqtt.handler.auth.plain.credentials-file=mqtt-credentials.csv
    ```
    * providing credentials as parameters
    ```
    mqtt-proxy server --mqtt.publisher.name=noop \
        --mqtt.handler.auth.name=plain \
        --mqtt.handler.auth.plain.credentials=alice=alice-secret \
        --mqtt.handler.auth.plain.credentials=bob=bob-secret
    ```
2. publish

    ```
    mosquitto_pub -m "on" -t "dummy" -u alice -P alice-secret
    mosquitto_pub -L mqtt://bob:bob-secret@localhost:1883/dummy -m "on"
    ```


## Configuration

### Kafka publisher

Kafka producer configuration properties used by [librdkafka](https://github.com/edenhill/librdkafka/blob/master/CONFIGURATION.md) should be prefixed with `producer.`

```
--mqtt.publisher.kafka.config=producer.sasl.mechanisms=PLAIN,producer.security.protocol=SASL_SSL,producer.sasl.username=myuser,producer.sasl.password=mypasswd
```

### Examples

- Ignore subscribe / unsubscribe requests

```
mqtt-proxy server  --mqtt.publisher.name=noop --mqtt.handler.ignore-unsupported SUBSCRIBE --mqtt.handler.ignore-unsupported UNSUBSCRIBE
```


## Metrics


metric | labels | description
-------| -------| ------------
|mqtt_proxy_build_info| branch, goversion, revision, revision|A metric with a constant '1' value labeled by version, revision, branch, and goversion from which mqtt_proxy was built.|
|mqtt_proxy_server_connections_active| |Number of active TCP connections from clients to server.|
|mqtt_proxy_server_connections_total| |Total number of TCP connections from clients to server.|
|mqtt_proxy_handler_requests_total| type, version |Total number of MQTT requests labeled by package control type and protocol version. |
|mqtt_proxy_handler_responses_total| type, version |Total number of MQTT responses labeled by package control type and protocol version. |
|mqtt_proxy_publisher_publish_duration_seconds | name, type, qos | Histogram tracking latencies for publish requests. |
|mqtt_proxy_authenticator_login_duration_seconds | name, code, err | Histogram tracking latencies for login requests. |
