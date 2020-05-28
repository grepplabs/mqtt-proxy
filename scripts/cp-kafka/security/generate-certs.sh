#!/usr/bin/env bash
PASSWORD="mqtt-proxy"
CN_HOST="localhost"
SERVER_KEYSTORE_JKS="docker.kafka.server.keystore.jks"
SERVER_KEYSTORE_P12="docker.kafka.server.keystore.p12"
SERVER_KEYSTORE_PEM="docker.kafka.server.keystore.pem"
SERVER_KEYSTORE_PASSWD="docker.kafka.server.keystore.passwd"
SERVER_KEYSTORE_PASS="mqtt-proxy-keystore"

SERVER_KEY_PASSWD="docker.kafka.server.key.passwd"

SERVER_TRUSTSTORE_JKS="docker.kafka.server.truststore.jks"
SERVER_TRUSTSTORE_PASSWD="docker.kafka.server.truststore.passwd"
SERVER_TRUSTSTORE_PASS="mqtt-proxy-truststore"

echo "Clearing existing Kafka SSL certs..."

BASEDIR=$(pwd)

mkdir -p ${BASEDIR}/certs
rm -rf ${BASEDIR}/certs/*

(
echo "Generating new Kafka SSL certs in \"${BASEDIR}/certs\" folder..."
cd ${BASEDIR}/certs

# https://docs.confluent.io/2.0.0/kafka/ssl.html

# Create keystore (-keyalg RSA)
keytool -keystore $SERVER_KEYSTORE_JKS -alias localhost -validity 7300 -keyalg RSA -genkey -storepass $SERVER_KEYSTORE_PASS -keypass $SERVER_KEYSTORE_PASS -dname "CN=${CN_HOST}, OU=None, O=grepplabs, L=Bonn, C=DE"

# Creating your own CA
openssl req -new -x509 -keyout ca-key.pem -out ca-cert.pem -days 7300 -passout pass:$SERVER_KEYSTORE_PASS -subj "/C=DE/L=Bonn/O=grepplabs/OU=None/CN=${CN_HOST}"
keytool -keystore $SERVER_TRUSTSTORE_JKS -alias CARoot -import -file ca-cert.pem -storepass $SERVER_TRUSTSTORE_PASS -noprompt

# Signing the certificate
# (openssl x509 -req never copies extensions from the CSR) keytool -keystore $SERVER_KEYSTORE_JKS -alias localhost -certreq -ext SAN=dns:localhost,dns:broker,ip:127.0.0.1 -file cert-req.pem -storepass $SERVER_KEYSTORE_PASS -noprompt
keytool -keystore $SERVER_KEYSTORE_JKS -alias localhost -certreq -file cert-req.pem -storepass $SERVER_KEYSTORE_PASS -noprompt
openssl x509 -req -CA ca-cert.pem -CAkey ca-key.pem -in cert-req.pem -out cert-signed.pem -days 7300 -extensions v3_req_broker -extfile $BASEDIR/openssl.conf -CAcreateserial -passin pass:$SERVER_KEYSTORE_PASS
keytool -keystore $SERVER_KEYSTORE_JKS -alias CARoot -import -file ca-cert.pem -storepass $SERVER_KEYSTORE_PASS -noprompt
keytool -keystore $SERVER_KEYSTORE_JKS -alias localhost -import -file cert-signed.pem -storepass $SERVER_KEYSTORE_PASS -noprompt

# Keystore as PEM
keytool -importkeystore -srckeystore $SERVER_KEYSTORE_JKS -destkeystore $SERVER_KEYSTORE_P12 -srcstoretype JKS -deststoretype PKCS12 -srcstorepass $SERVER_KEYSTORE_PASS -deststorepass $SERVER_KEYSTORE_PASS -noprompt
openssl pkcs12 -in $SERVER_KEYSTORE_P12 -out $SERVER_KEYSTORE_PEM -nodes -passin pass:$SERVER_KEYSTORE_PASS

# Passwords files
echo -n "$SERVER_KEYSTORE_PASS" > ${SERVER_KEYSTORE_PASSWD}
echo -n "$SERVER_KEYSTORE_PASS" > ${SERVER_KEY_PASSWD}
echo -n "$SERVER_TRUSTSTORE_PASS" > ${SERVER_TRUSTSTORE_PASSWD}

# mqtt proxy certs
openssl req -new -newkey rsa:2048 -config $BASEDIR/openssl.conf -nodes -keyout proxy-key.pem -out proxy-req.pem
openssl x509 -req -CA ca-cert.pem -CAkey ca-key.pem -in proxy-req.pem -out proxy-signed.pem -days 7300 -extensions v3_req_proxy -extfile $BASEDIR/openssl.conf -CAcreateserial -passin pass:$SERVER_KEYSTORE_PASS

chmod +rx *
)