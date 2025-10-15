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
  default     = "multi-container-alb"
}

variable "service_name" {
  description = "ECS service name"
  type        = string
  default     = "multi-container-alb"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "development"
}

variable "vpc_id" {
  description = "VPC ID for Target Group"
  type        = string
  default     = "vpc-12345678"
}

variable "subnets" {
  description = "Subnets for ALB"
  type        = list(string)
  default     = ["subnet-12345678", "subnet-87654321"]
}

variable "security_groups" {
  description = "Security groups for ALB"
  type        = list(string)
  default     = ["sg-webapp"]
}
