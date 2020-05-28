.DEFAULT_GOAL := build-up
.PHONY: build up down build-up

ROOT_DIR       := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
MQTT_PROXY     := $(ROOT_DIR)/../../mqtt-proxy

build-up: build up

build:
	docker-compose build

up:
	docker-compose up

up-d:
	docker-compose up -d

down:
	docker-compose down

test-publish:
	docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 0" --repeat 1 -q 0
	docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 1" --repeat 1 -q 1
	docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 2" --repeat 1 -q 2

test-listen:
	docker exec -it broker kafka-console-consumer --bootstrap-server localhost:9092 --topic mqtt-test --property print.key=true --from-beginning

server-security-plaintext:
	@$(MQTT_PROXY) server \
       --mqtt.listen-address="0.0.0.0:2883" \
       --http.listen-address="0.0.0.0:10090" \
       --mqtt.publisher.name=kafka \
	   --mqtt.publisher.kafka.bootstrap-servers=localhost:9092 \
	   --mqtt.publisher.kafka.default-topic=mqtt-test

server-security-sasl-plaintext:
	@$(MQTT_PROXY) server \
       --mqtt.listen-address="0.0.0.0:2883" \
       --http.listen-address="0.0.0.0:10090" \
       --mqtt.publisher.name=kafka \
	   --mqtt.publisher.kafka.bootstrap-servers=localhost:9093 \
	   --mqtt.publisher.kafka.config=producer.sasl.mechanisms=PLAIN,producer.security.protocol=SASL_PLAINTEXT,producer.sasl.username=mqtt_proxy,producer.sasl.password=mqtt-proxy-secret \
	   --mqtt.publisher.kafka.default-topic=mqtt-test

server-security-ssl:
	@$(MQTT_PROXY) server \
       --mqtt.listen-address="0.0.0.0:2883" \
       --http.listen-address="0.0.0.0:10090" \
       --mqtt.publisher.name=kafka \
	   --mqtt.publisher.kafka.bootstrap-servers=localhost:9094 \
	   --mqtt.publisher.kafka.config=producer.security.protocol=SSL,producer.ssl.ca.location=$(ROOT_DIR)/security/certs/ca-cert.pem \
	   --mqtt.publisher.kafka.default-topic=mqtt-test

server-security-sasl-ssl:
	@$(MQTT_PROXY) server \
       --mqtt.listen-address="0.0.0.0:2883" \
       --http.listen-address="0.0.0.0:10090" \
       --mqtt.publisher.name=kafka \
	   --mqtt.publisher.kafka.bootstrap-servers=localhost:9095 \
	   --mqtt.publisher.kafka.config=producer.sasl.mechanisms=PLAIN,producer.security.protocol=SASL_SSL,producer.ssl.ca.location=$(ROOT_DIR)/security/certs/ca-cert.pem,producer.sasl.username=mqtt_proxy,producer.sasl.password=mqtt-proxy-secret \
	   --mqtt.publisher.kafka.default-topic=mqtt-test

server-tls-security-plaintext:
	@$(MQTT_PROXY) server \
       --mqtt.listen-address="0.0.0.0:3883" \
       --http.listen-address="0.0.0.0:20090" \
       --mqtt.server-tls-key=$(ROOT_DIR)/security/certs/proxy-key.pem  \
       --mqtt.server-tls-cert=$(ROOT_DIR)/security/certs/proxy-signed.pem  \
       --mqtt.publisher.name=kafka \
	   --mqtt.publisher.kafka.bootstrap-servers=localhost:9092 \
	   --mqtt.publisher.kafka.default-topic=mqtt-test

publish-mosquitto:
	mosquitto_pub -L mqtt://localhost:2883/dummy -m "test qos 0" --repeat 1 -q 0
	mosquitto_pub -L mqtt://localhost:2883/dummy -m "test qos 1" --repeat 1 -q 1
	mosquitto_pub -L mqtt://localhost:2883/dummy -m "test qos 2" --repeat 1 -q 2
