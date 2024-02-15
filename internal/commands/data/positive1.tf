resource "aws_lb_listener" "listener5" {
  load_balancer_arn = aws_lb.test3.arn
  port = 80
  default_action {
    type = "redirect"

    redirect {
      port        = "80"
      protocol    = "HTTPSSSSSSSSSSSSSSSSSSSSSSSSS"
      status_code = "HTTP_301"
    }
  }
}

resource "aws_lb" "test3" {
  enable_deletion_protection = true
  drop_invalid_header_fields = true
  name = "test123"
  load_balancer_type = "application"
  subnets = [aws_subnet.subnet1.id, aws_subnet.subnet2.id]
  internal = true
}
