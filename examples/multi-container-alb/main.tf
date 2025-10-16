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

# ECS Task Definition
resource "aws_ecs_task_definition" "main" {
  family                   = var.service_name
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "512"
  memory                   = "1024"
  execution_role_arn       = "arn:aws:iam::000000000000:role/ecsTaskExecutionRole"

  volume {
    name = "shared-data"
  }

  container_definitions = jsonencode([
    {
      name      = "frontend-nginx"
      image     = "nginx:alpine"
      essential = true
      portMappings = [
        {
          containerPort = 80
          protocol      = "tcp"
        }
      ]
      environment = [
        {
          name  = "API_ENDPOINT"
          value = "http://localhost:3000"
        }
      ]
      mountPoints = [
        {
          sourceVolume  = "shared-data"
          containerPath = "/usr/share/nginx/html"
          readOnly      = true
        }
      ]
      dependsOn = [
        {
          containerName = "backend-api"
          condition     = "HEALTHY"
        }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.service.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "frontend"
        }
      }
    },
    {
      name      = "backend-api"
      image     = "node:18-alpine"
      essential = true
      portMappings = [
        {
          containerPort = 3000
          protocol      = "tcp"
        }
      ]
      command = [
        "sh",
        "-c",
        "echo '{\"status\":\"ok\",\"message\":\"API Running\"}' > /data/status.json && node -e 'require(\"http\").createServer((req,res)=>{res.writeHead(200,{\"Content-Type\":\"application/json\"});res.end(JSON.stringify({status:\"healthy\",timestamp:new Date()}))}).listen(3000,()=>console.log(\"API running on :3000\"))'"
      ]
      mountPoints = [
        {
          sourceVolume  = "shared-data"
          containerPath = "/data"
        }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.service.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "backend"
        }
      }
      healthCheck = {
        command     = ["CMD-SHELL", "wget -q -O - http://localhost:3000 || exit 1"]
        interval    = 30
        timeout     = 5
        retries     = 3
        startPeriod = 30
      }
    },
    {
      name      = "sidecar-logger"
      image     = "busybox:latest"
      essential = false
      command = [
        "sh",
        "-c",
        "while true; do echo \"[$(date)] Health check performed\" >> /data/health.log; sleep 60; done"
      ]
      mountPoints = [
        {
          sourceVolume  = "shared-data"
          containerPath = "/data"
        }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.service.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "sidecar"
        }
      }
    }
  ])

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# ECS Service
resource "aws_ecs_service" "main" {
  name            = var.service_name
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.main.arn
  desired_count   = 3
  launch_type     = "FARGATE"
  platform_version = "LATEST"

  network_configuration {
    subnets          = var.subnets
    security_groups  = var.security_groups
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.main.arn
    container_name   = "frontend-nginx"
    container_port   = 80
  }

  health_check_grace_period_seconds = 60
  scheduling_strategy               = "REPLICA"

  enable_ecs_managed_tags = true
  propagate_tags          = "TASK_DEFINITION"
  enable_execute_command  = false

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }

  depends_on = [
    aws_lb_listener.http
  ]
}
