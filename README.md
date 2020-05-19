# mqtt-proxy

**Work in progress**


## Metrics


metric | labels | description
-------| -------| ------------
|mqtt_proxy_build_info| branch, goversion, revision, revision|A metric with a constant '1' value labeled by version, revision, branch, and goversion from which mqtt_proxy was built.|
|mqtt_proxy_server_connections_active| |Number of active TCP connections from clients to server.|
|mqtt_proxy_server_connections_total| |Total number of TCP connections from clients to server.|
|mqtt_proxy_handler_requests_total|type|Total number of MQTT requests labeled by package control type. |
|mqtt_proxy_handler_responses_total|type|Total number of MQTT responses labeled by package control type. |