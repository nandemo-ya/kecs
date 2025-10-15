terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.aws_region

  # KECS endpoint configuration
  endpoints {
    ecs            = var.kecs_endpoint
    secretsmanager = var.kecs_endpoint
    ssm            = var.kecs_endpoint
    logs           = var.kecs_endpoint
    iam            = var.kecs_endpoint
  }

  # Skip credential validation for local KECS environment
  skip_credentials_validation = true
  skip_requesting_account_id  = true
  skip_metadata_api_check     = true

  # Use fake credentials for KECS
  access_key = "test"
  secret_key = "test"
}
