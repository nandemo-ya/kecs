output "cluster_name" {
  description = "ECS cluster name"
  value       = aws_ecs_cluster.main.name
}

output "cluster_arn" {
  description = "ECS cluster ARN"
  value       = aws_ecs_cluster.main.arn
}

output "log_group_name" {
  description = "CloudWatch Logs log group name"
  value       = aws_cloudwatch_log_group.nginx_logs.name
}

output "log_group_arn" {
  description = "CloudWatch Logs log group ARN"
  value       = aws_cloudwatch_log_group.nginx_logs.arn
}
