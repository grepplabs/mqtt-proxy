.DEFAULT_GOAL := build-up
.PHONY: build up down build-up

# ROOT_DIR       := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
ROOT_DIR       := .
MQTT_PROXY     := $(ROOT_DIR)/../../mqtt-proxy
CFSSL_DIR      := $(ROOT_DIR)/security/cfssl

.PHONY: help
help:
	@grep -E '^[a-zA-Z%_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build-up: build up

build: ## docker-compose build
	docker-compose build

up: ## docker-compose up
	docker-compose up

up-d: ## docker-compose up in detached mode
	docker-compose up -d

down: ## docker-compose down
	docker-compose down

test-publish:  ## test publish using mosquitto client
	docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 0" --repeat 1 -q 0
	docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 1" --repeat 1 -q 1
	docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 2" --repeat 1 -q 2
	docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 0" --repeat 1 -q 0 -V mqttv5
	docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 1" --repeat 1 -q 1 -V mqttv5
	docker exec -it mqtt-client mosquitto_pub -L mqtt://mqtt-proxy:1883/dummy -m "test qos 2" --repeat 1 -q 2 -V mqttv5

test-listen: ## start kafka-console-consumer
	docker exec -it broker kafka-console-consumer --bootstrap-server localhost:9092 --topic mqtt-test --property print.key=true --from-beginning

### PLAIN listener

server-security-plaintext: ## start mqtt-proxy
	@$(MQTT_PROXY) server \
       --mqtt.listen-address="0.0.0.0:2883" \
       --http.listen-address="0.0.0.0:10090" \
       --mqtt.publisher.name=kafka \
	   --mqtt.publisher.kafka.bootstrap-servers=localhost:9092 \
	   --mqtt.publisher.kafka.default-topic=mqtt-test

server-security-sasl-plaintext: ## start mqtt-proxy with kafka producer using SASL_PLAINTEXT
	@$(MQTT_PROXY) server \
       --mqtt.listen-address="0.0.0.0:2883" \
       --http.listen-address="0.0.0.0:10090" \
       --mqtt.publisher.name=kafka \
	   --mqtt.publisher.kafka.bootstrap-servers=localhost:9093 \
	   --mqtt.publisher.kafka.config=producer.sasl.mechanisms=PLAIN,producer.security.protocol=SASL_PLAINTEXT,producer.sasl.username=mqtt_proxy,producer.sasl.password=mqtt-proxy-secret \
	   --mqtt.publisher.kafka.default-topic=mqtt-test

server-security-ssl: ## start mqtt-proxy with kafka producer using SSL
	@$(MQTT_PROXY) server \
       --mqtt.listen-address="0.0.0.0:2883" \
       --http.listen-address="0.0.0.0:10090" \
       --mqtt.publisher.name=kafka \
	   --mqtt.publisher.kafka.bootstrap-servers=localhost:9094 \
	   --mqtt.publisher.kafka.config=producer.security.protocol=SSL,producer.ssl.ca.location=$(ROOT_DIR)/security/certs/ca-cert.pem \
	   --mqtt.publisher.kafka.default-topic=mqtt-test

server-security-sasl-ssl: ## start mqtt-proxy with kafka producer using SASL_SSL
	@$(MQTT_PROXY) server \
       --mqtt.listen-address="0.0.0.0:2883" \
       --http.listen-address="0.0.0.0:10090" \
       --mqtt.publisher.name=kafka \
	   --mqtt.publisher.kafka.bootstrap-servers=localhost:9095 \
	   --mqtt.publisher.kafka.config=producer.sasl.mechanisms=PLAIN,producer.security.protocol=SASL_SSL,producer.ssl.ca.location=$(ROOT_DIR)/security/certs/ca-cert.pem,producer.sasl.username=mqtt_proxy,producer.sasl.password=mqtt-proxy-secret \
	   --mqtt.publisher.kafka.default-topic=mqtt-test


publish:
	mosquitto_pub -L mqtt://localhost:2883/dummy -m "test qos 0" --repeat 1 -q 0
	mosquitto_pub -L mqtt://localhost:2883/dummy -m "test qos 1" --repeat 1 -q 1
	mosquitto_pub -L mqtt://localhost:2883/dummy -m "test qos 2" --repeat 1 -q 2
	mosquitto_pub -L mqtt://localhost:2883/dummy -m "test qos 0" --repeat 1 -q 0 -V mqttv5
	mosquitto_pub -L mqtt://localhost:2883/dummy -m "test qos 1" --repeat 1 -q 1 -V mqttv5
	mosquitto_pub -L mqtt://localhost:2883/dummy -m "test qos 2" --repeat 1 -q 2 -V mqttv5

### TLS listener
server-tls-generate-certs:
	$(shell bash -c "cd $(CFSSL_DIR) && earthly +hosts")

server-tls-force-generate-certs:
	$(shell bash -c "cd $(CFSSL_DIR) && earthly --no-cache +hosts")

server-tls: ## start mqtt-proxy with TLS listener
	@$(MQTT_PROXY) server \
       --mqtt.listen-address="0.0.0.0:3883" \
       --http.listen-address="0.0.0.0:20090" \
       --mqtt.server-tls.enable \
       --mqtt.server-tls.refresh 1s \
       --mqtt.server-tls.file.key=$(CFSSL_DIR)/mqtt-proxy-server-l2-key.pem  \
       --mqtt.server-tls.file.cert=$(CFSSL_DIR)/mqtt-proxy-server-l2-l1.pem  \
       --mqtt.publisher.name=kafka \
	   --mqtt.publisher.kafka.bootstrap-servers=localhost:9092 \
	   --mqtt.publisher.kafka.default-topic=mqtt-test

server-tls-auth: ## start mqtt-proxy with TLS listener with client auth
	@$(MQTT_PROXY) server \
       --mqtt.listen-address="0.0.0.0:3883" \
       --http.listen-address="0.0.0.0:20090" \
       --mqtt.server-tls.enable \
       --mqtt.server-tls.refresh 1s \
       --mqtt.server-tls.file.key=$(CFSSL_DIR)/mqtt-proxy-server-l2-key.pem  \
       --mqtt.server-tls.file.cert=$(CFSSL_DIR)/mqtt-proxy-server-l2-l1.pem  \
       --mqtt.server-tls.file.client-ca=$(CFSSL_DIR)/intermediate_ca.pem  \
       --mqtt.publisher.name=kafka \
	   --mqtt.publisher.kafka.bootstrap-servers=localhost:9092 \
	   --mqtt.publisher.kafka.default-topic=mqtt-test

server-tls-auth-clr: ## start mqtt-proxy with TLS listener with client auth with CRL config
	@$(MQTT_PROXY) server \
       --mqtt.listen-address="0.0.0.0:3883" \
       --http.listen-address="0.0.0.0:20090" \
       --log.level=debug \
       --mqtt.server-tls.enable \
       --mqtt.server-tls.refresh 1s \
       --mqtt.server-tls.file.key=$(CFSSL_DIR)/mqtt-proxy-server-l2-key.pem  \
       --mqtt.server-tls.file.cert=$(CFSSL_DIR)/mqtt-proxy-server-l2-l1.pem  \
       --mqtt.server-tls.file.client-ca=$(CFSSL_DIR)/intermediate_ca.pem  \
       --mqtt.server-tls.file.client-clr=$(CFSSL_DIR)/intermediate-crl.pem \
       --mqtt.publisher.name=kafka \
	   --mqtt.publisher.kafka.bootstrap-servers=localhost:9092 \
	   --mqtt.publisher.kafka.default-topic=mqtt-test

publish-tls:
	mosquitto_pub -L mqtts://localhost:3883/dummy --cafile $(CFSSL_DIR)/ca.pem -m "test qos 0" --repeat 1 -q 0
	mosquitto_pub -L mqtts://localhost:3883/dummy --cafile $(CFSSL_DIR)/ca.pem -m "test qos 1" --repeat 1 -q 1
	mosquitto_pub -L mqtts://localhost:3883/dummy --cafile $(CFSSL_DIR)/chain_ca-l2-l1.pem -m "test qos 2" --repeat 1 -q 2

publish-tls-auth:
	mosquitto_pub -L mqtts://localhost:3883/dummy --cafile $(CFSSL_DIR)/ca.pem --key $(CFSSL_DIR)/mqtt-proxy-client-1-l2-key.pem --cert $(CFSSL_DIR)/mqtt-proxy-client-1-l2.pem -m "test qos 0" --repeat 1 -q 0
	mosquitto_pub -L mqtts://localhost:3883/dummy --cafile $(CFSSL_DIR)/ca.pem --key $(CFSSL_DIR)/mqtt-proxy-client-2-l2-key.pem --cert $(CFSSL_DIR)/mqtt-proxy-client-2-l2.pem -m "test qos 0" --repeat 1 -q 0
