.DEFAULT_GOAL := build-up
.PHONY: build up down build-up

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
