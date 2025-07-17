# outputs
output "aws_region" {
  description = "aws region"
  value       = var.aws_region
}

output "environment" {
  description = "environment name"
  value       = var.environment
}

output "account_id" {
  description = "aws account id"
  value       = data.aws_caller_identity.current.account_id
}

output "db_endpoint" {
  description = "rds postgres endpoint"
  value       = aws_db_instance.default.endpoint
}

output "db_port" {
  description = "rds postgres port" 
  value       = aws_db_instance.default.port
}

output "transactions_queue_url" {
  value = aws_sqs_queue.transactions_queue.url
}

output "notifications_queue_url" {
  value = aws_sqs_queue.notifications_queue.url
}