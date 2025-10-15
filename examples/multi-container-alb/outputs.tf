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
  value       = aws_cloudwatch_log_group.service.name
}

output "log_group_arn" {
  description = "CloudWatch Logs log group ARN"
  value       = aws_cloudwatch_log_group.service.arn
}

output "alb_name" {
  description = "Application Load Balancer name"
  value       = aws_lb.main.name
}

output "alb_arn" {
  description = "Application Load Balancer ARN"
  value       = aws_lb.main.arn
}

output "alb_dns_name" {
  description = "Application Load Balancer DNS name"
  value       = aws_lb.main.dns_name
}

output "target_group_name" {
  description = "Target Group name"
  value       = aws_lb_target_group.main.name
}

output "target_group_arn" {
  description = "Target Group ARN (use this in service_def_with_elb.json)"
  value       = aws_lb_target_group.main.arn
}

output "listener_arn" {
  description = "HTTP Listener ARN"
  value       = aws_lb_listener.http.arn
}
