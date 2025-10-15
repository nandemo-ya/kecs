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

# Application Load Balancer
resource "aws_lb" "main" {
  name               = "${var.service_name}-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = var.security_groups
  subnets            = var.subnets
  ip_address_type    = "ipv4"

  tags = {
    Application = var.service_name
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Target Group
resource "aws_lb_target_group" "main" {
  name        = "${var.service_name}-tg"
  port        = 80
  protocol    = "HTTP"
  vpc_id      = var.vpc_id
  target_type = "ip"

  health_check {
    enabled             = true
    path                = "/"
    interval            = 30
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 3
    matcher             = "200,301,302,404"
  }

  tags = {
    Application = var.service_name
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# HTTP Listener
resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.main.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.main.arn
  }

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Listener Rule for API endpoints
resource "aws_lb_listener_rule" "api" {
  listener_arn = aws_lb_listener.http.arn
  priority     = 1

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.main.arn
  }

  condition {
    path_pattern {
      values = ["/api/*"]
    }
  }

  tags = {
    Name        = "api-route"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Listener Rule for static assets
resource "aws_lb_listener_rule" "static" {
  listener_arn = aws_lb_listener.http.arn
  priority     = 2

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.main.arn
  }

  condition {
    path_pattern {
      values = ["/static/*"]
    }
  }

  tags = {
    Name        = "static-route"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# Listener Rule for health checks
resource "aws_lb_listener_rule" "health" {
  listener_arn = aws_lb_listener.http.arn
  priority     = 3

  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.main.arn
  }

  condition {
    path_pattern {
      values = ["/health"]
    }
  }

  tags = {
    Name        = "health-route"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}
