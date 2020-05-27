data "template_file" "mqtt-proxy-init" {
  template = file("${path.module}/proxy.tpl")

  vars = {
    mqtt_proxy_version = var.mqtt_proxy_version
    kafka_proxy_version = var.kafka_proxy_version
    bootstrap_servers = aws_msk_cluster.mqtt-proxy-cluster.bootstrap_brokers
  }
}

resource "aws_instance" "mqtt-proxy" {
  count                  = var.mqtt_proxy_enable ? 1 : 0
  ami                    = data.aws_ami.ubuntu-focal.id
  instance_type          = var.mqtt_proxy_ec2_instance_type
  subnet_id              = data.aws_subnet.subnets.0.id
  iam_instance_profile   = aws_iam_instance_profile.mqtt-proxy-profile.id
  vpc_security_group_ids = [aws_security_group.mqtt-proxy-security-group.id]
  key_name               = aws_key_pair.mqtt-proxy-key-pair.key_name
  user_data              = data.template_file.mqtt-proxy-init.rendered
}

data "aws_ami" "ubuntu-focal" {
  most_recent = true

  filter {
    name = "name"
    values = [
      "*ubuntu-focal-*"]
  }
  filter {
    name = "architecture"
    values = [
      "x86_64"]
  }
  filter {
    name = "virtualization-type"
    values = [
      "hvm"]
  }
  filter {
    name = "root-device-type"
    values = [
      "ebs"]
  }
  owners = [
    "099720109477"]
}

resource "aws_key_pair" "mqtt-proxy-key-pair" {
  key_name   = "mqtt-proxy-key"
  public_key = var.mqtt_proxy_ec2_public_key
}

resource "aws_iam_instance_profile" "mqtt-proxy-profile" {
  name = "mqtt-proxy-instance-profile"
  role = aws_iam_role.mqtt-proxy-role.name
}

resource "aws_iam_role" "mqtt-proxy-role" {
  name = "mqtt-proxy-role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow"
    }
  ]
}
EOF
}

resource "aws_security_group" "mqtt-proxy-security-group" {
  name   = "mqtt-proxy-security-group"
  vpc_id = data.aws_vpc.vpc.id

  ingress {
    from_port = 1883
    to_port = 1883
    protocol = "tcp"
    cidr_blocks = [
      "0.0.0.0/0"]
  }
  ingress {
    from_port = 8883
    to_port = 8883
    protocol = "tcp"
    cidr_blocks = [
      "0.0.0.0/0"]
  }
  ingress {
    from_port = 9090
    to_port = 9090
    protocol = "tcp"
    cidr_blocks = [
      "0.0.0.0/0"]
  }
  ingress {
    from_port = 22
    to_port = 22
    protocol = "tcp"
    cidr_blocks = [
      "0.0.0.0/0"]
  }

  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = [
      "0.0.0.0/0"]
  }
}

output "mqtt_proxy_ip" {
  value = var.mqtt_proxy_enable ? aws_instance.mqtt-proxy.0.public_ip : ""
}
