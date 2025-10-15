# ECS Cluster
resource "aws_ecs_cluster" "main" {
  name = var.cluster_name

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# CloudWatch Logs Log Group
resource "aws_cloudwatch_log_group" "nginx" {
  name              = "/ecs/${var.service_name}"
  retention_in_days = 7

  tags = {
    Environment = var.environment
    Service     = var.service_name
    ManagedBy   = "terraform"
  }
}
