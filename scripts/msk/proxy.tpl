#!/usr/bin/env bash
curl -Ls https://github.com/grepplabs/mqtt-proxy/releases/download/${mqtt_proxy_version}/mqtt-proxy-${mqtt_proxy_version}-linux-amd64.tar.gz | tar xz
mv ./mqtt-proxy /usr/local/bin/mqtt-proxy

# kafka-proxy is not required by mqtt-proxy
curl -Ls https://github.com/grepplabs/kafka-proxy/releases/download/${kafka_proxy_version}/kafka-proxy-${kafka_proxy_version}-linux-amd64.tar.gz | tar xz
mv ./kafka-proxy /usr/local/bin/kafka-proxy

# run mqtt-proxy in podman
. /etc/os-release
sh -c "echo 'deb https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/xUbuntu_$${VERSION_ID}/ /' > /etc/apt/sources.list.d/devel:kubic:libcontainers:stable.list"
curl -L https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/xUbuntu_$${VERSION_ID}/Release.key | apt-key add -
apt-get update -qq
apt-get -qq -y install podman

mkdir -p /mqtt-proxy

tee /mqtt-proxy/mqtt-proxy.yml <<POD_FILE
---
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: mqtt-proxy
  name: mqtt-proxy
spec:
  containers:
  - command:
    - server
    - --mqtt.publisher.name=kafka
    - --mqtt.publisher.kafka.bootstrap-servers=${bootstrap_servers}
    - --mqtt.publisher.kafka.default-topic=mqtt-test
    env:
    - name: HOSTNAME
    - name: container
      value: podman
    image: docker.io/grepplabs/mqtt-proxy:latest
    name: mqtt-proxy
    ports:
    - containerPort: 9090
      hostPort: 9090
      protocol: TCP
    - containerPort: 1883
      hostPort: 1883
      protocol: TCP

POD_FILE

tee /etc/systemd/system/mqtt-proxy.service <<SYSTEMD_FILE
[Unit]
Description=MQTT Proxy

[Service]
Restart=always
ExecStartPre=/usr/bin/podman pod rm -i -f mqtt-proxy_pod
ExecStartPre=/usr/bin/podman rm -i -f mqtt-proxy
ExecStart=/usr/bin/podman play kube /mqtt-proxy/mqtt-proxy.yml
ExecStop=/usr/bin/podman stop -t 10 mqtt-proxy
KillMode=none
Type=forking

[Install]
WantedBy=multi-user.target

SYSTEMD_FILE

systemctl daemon-reload
systemctl start mqtt-proxy
systemctl status mqtt-proxy
systemctl enable mqtt-proxy
