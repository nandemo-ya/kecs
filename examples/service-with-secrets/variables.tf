variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "kecs_endpoint" {
  description = "KECS endpoint URL"
  type        = string
  default     = "http://localhost:5373"
}

variable "cluster_name" {
  description = "ECS cluster name"
  type        = string
  default     = "service-with-secrets"
}

variable "service_name" {
  description = "Service name"
  type        = string
  default     = "service-with-secrets"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "prod"
}

variable "app_name" {
  description = "Application name"
  type        = string
  default     = "myapp"
}
