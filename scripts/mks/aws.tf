terraform {
  required_version = ">= 0.12.18"
}

provider "aws" {
  region = var.region
  version = ">= 2.45.0"
}

data "aws_caller_identity" "current" {}

data "aws_region" "current" {}

data "aws_availability_zones" "available" {}

data "aws_vpc" "vpc" {
  filter {
    name = "tag:Name"
    values = [
      "default"
    ]
  }
}

data "aws_subnet_ids" "subnets" {
  vpc_id = data.aws_vpc.vpc.id
}


data "aws_subnet" "subnets" {
  count = length(data.aws_subnet_ids.subnets.ids)
  id    = tolist(data.aws_subnet_ids.subnets.ids)[count.index]
}
