# ECS Cluster
resource "aws_ecs_cluster" "main" {
  name = var.cluster_name

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# CloudWatch Logs Log Group
resource "aws_cloudwatch_log_group" "service" {
  name              = "/ecs/${var.service_name}"
  retention_in_days = 7

  tags = {
    Environment = var.environment
    Service     = var.service_name
    ManagedBy   = "terraform"
  }
}

# SSM Parameters
resource "aws_ssm_parameter" "database_url" {
  name        = "/${var.app_name}/${var.environment}/database-url"
  description = "Production database connection string"
  type        = "SecureString"
  value       = "postgresql://app_user:password@db.example.com:5432/${var.app_name}"

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_ssm_parameter" "api_key" {
  name        = "/${var.app_name}/${var.environment}/api-key"
  description = "Production API key"
  type        = "SecureString"
  value       = "sk_live_abcdef123456789"

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_ssm_parameter" "feature_flags" {
  name        = "/${var.app_name}/${var.environment}/feature-flags"
  description = "Feature flags configuration"
  type        = "String"
  value       = jsonencode({
    new_ui           = true
    beta_features    = false
    maintenance_mode = false
  })

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Secrets Manager Secrets
resource "aws_secretsmanager_secret" "db" {
  name        = "${var.app_name}/${var.environment}/db"
  description = "Database credentials"

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_secretsmanager_secret_version" "db" {
  secret_id = aws_secretsmanager_secret.db.id
  secret_string = jsonencode({
    username = "admin"
    password = "super-secret-password"
  })
}

resource "aws_secretsmanager_secret" "jwt" {
  name        = "${var.app_name}/${var.environment}/jwt"
  description = "JWT signing key"

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_secretsmanager_secret_version" "jwt" {
  secret_id = aws_secretsmanager_secret.jwt.id
  secret_string = jsonencode({
    key = "very-secret-jwt-key-12345"
  })
}

resource "aws_secretsmanager_secret" "encryption" {
  name        = "${var.app_name}/${var.environment}/encryption"
  description = "Encryption keys"

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

resource "aws_secretsmanager_secret_version" "encryption" {
  secret_id = aws_secretsmanager_secret.encryption.id
  secret_string = jsonencode({
    aes_key = "256bit-encryption-key-example"
  })
}
