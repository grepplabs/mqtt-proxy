resource "aws_security_group" "mqtt-proxy-cluster-security-group" {
  vpc_id = data.aws_vpc.vpc.id

  ingress {
    from_port       = 9092
    to_port         = 9092
    protocol        = "tcp"
    security_groups = [
      aws_security_group.mqtt-proxy-security-group.id
    ]
  }
  ingress {
    from_port       = 9094
    to_port         = 9094
    protocol        = "tcp"
    security_groups = [
      aws_security_group.mqtt-proxy-security-group.id
    ]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = [
      "0.0.0.0/0"
    ]
  }
}

resource "aws_security_group_rule" "all-ingress-traffic" {
  count             = var.public_access ? 1 : 0
  from_port         = 0
  to_port           = 0
  protocol          = "all"
  cidr_blocks       = ["0.0.0.0/0"]
  ipv6_cidr_blocks  = ["::/0"]
  type              = "ingress"
  security_group_id = aws_security_group.mqtt-proxy-cluster-security-group.id
}

resource "aws_msk_cluster" "mqtt-proxy-cluster" {
  cluster_name           = "mqtt-proxy-cluster"
  kafka_version          = var.kafka_version
  number_of_broker_nodes = var.kafka_number_of_broker_nodes

  broker_node_group_info {
    instance_type   = var.kafka_broker_instance_type
    client_subnets  = [for subnet in data.aws_subnet.subnets : subnet.id]
    security_groups = [aws_security_group.mqtt-proxy-cluster-security-group.id]
    storage_info {
      ebs_storage_info {
        volume_size = var.kafka_broker_ebs_volume_size
      }
    }
    connectivity_info {
      public_access {
        type = var.public_access ? "SERVICE_PROVIDED_EIPS" : "DISABLED"
      }
    }
  }
  dynamic "client_authentication" {
    for_each = var.sasl_iam_enable ? [1] : []
    content {
      sasl {
        iam   = true
        scram = false
      }
    }
  }
  encryption_info {
    encryption_in_transit {
      client_broker = var.encryption_client_broker
    }
  }
}

output "zookeeper_connect_string" {
  value = aws_msk_cluster.mqtt-proxy-cluster.zookeeper_connect_string
}

output "bootstrap_brokers" {
  value = aws_msk_cluster.mqtt-proxy-cluster.bootstrap_brokers
}

output "bootstrap_brokers_public_sasl_iam" {
  value = aws_msk_cluster.mqtt-proxy-cluster.bootstrap_brokers_public_sasl_iam
}

output "bootstrap_brokers_public_sasl_scram" {
  value = aws_msk_cluster.mqtt-proxy-cluster.bootstrap_brokers_public_sasl_scram
}
output "bootstrap_brokers_public_tls" {
  value = aws_msk_cluster.mqtt-proxy-cluster.bootstrap_brokers_public_tls
}

output "bootstrap_brokers_sasl_scram" {
  value = aws_msk_cluster.mqtt-proxy-cluster.bootstrap_brokers_sasl_scram
}

output "bootstrap_brokers_tls" {
  value = aws_msk_cluster.mqtt-proxy-cluster.bootstrap_brokers_tls
}
