variable "region" {
  type    = string
  default = "eu-central-1"
}

variable "mqtt_proxy_version" {
  type = string
}

variable "mqtt_proxy_ec2_public_key" {
  type = string
}

variable "mqtt_proxy_ec2_instance_type" {
  type = string
}

variable "mqtt_proxy_enable" {
  type    = bool
  default = true
}

variable "kafka_proxy_version" {
  type = string
}

variable "kafka_version" {
  type    = string
  default = "2.8.1"
}

variable "kafka_number_of_broker_nodes" {
  type    = number
  default = 3
}

variable "kafka_broker_instance_type" {
  type = string
}

variable "kafka_broker_ebs_volume_size" {
  type = number
}

variable "sasl_iam_enable" {
  type    = bool
  default = false
}

variable "public_access" {
  type    = bool
  default = false
}

variable "encryption_client_broker" {
  type = string
  default = "TLS_PLAINTEXT"
}