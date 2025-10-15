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
  default     = "nginx-with-log-generator"
}

variable "service_name" {
  description = "ECS service name"
  type        = string
  default     = "nginx-with-log-generator"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "development"
}
