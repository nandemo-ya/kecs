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

output "ssm_parameters" {
  description = "SSM parameter names"
  value = {
    database_url  = aws_ssm_parameter.database_url.name
    api_key       = aws_ssm_parameter.api_key.name
    feature_flags = aws_ssm_parameter.feature_flags.name
  }
}

output "secrets_manager_secrets" {
  description = "Secrets Manager secret ARNs"
  value = {
    db         = aws_secretsmanager_secret.db.arn
    jwt        = aws_secretsmanager_secret.jwt.arn
    encryption = aws_secretsmanager_secret.encryption.arn
  }
  sensitive = true
}

output "secret_arns_for_task_definition" {
  description = "Secret ARNs to use in ECS task definition"
  value = {
    db_arn         = aws_secretsmanager_secret.db.arn
    jwt_arn        = aws_secretsmanager_secret.jwt.arn
    encryption_arn = aws_secretsmanager_secret.encryption.arn
  }
}

output "ssm_parameter_arns_for_task_definition" {
  description = "SSM parameter ARNs to use in ECS task definition"
  value = {
    database_url_arn  = aws_ssm_parameter.database_url.arn
    api_key_arn       = aws_ssm_parameter.api_key.arn
    feature_flags_arn = aws_ssm_parameter.feature_flags.arn
  }
}
