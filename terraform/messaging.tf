resource "aws_sqs_queue" "transactions_queue" {
  name                      = "finsys-transactions-queue"
  max_message_size          = 2048
  message_retention_seconds = 86400
  receive_wait_time_seconds = 10
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.transactions_queue_deadletter.arn
    maxReceiveCount     = 4
  })

  tags = {
    Environment = var.environment
  }
}

resource "aws_sqs_queue" "notifications_queue" {
  name                      = "finsys-notifications-queue"
  max_message_size          = 2048
  message_retention_seconds = 86400
  receive_wait_time_seconds = 10
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.notifications_queue_deadletter.arn
    maxReceiveCount     = 4
  })

  tags = {
    Environment = var.environment
  }
}

resource "aws_sqs_queue" "transactions_queue_deadletter" {
  name = "finsys-transactions-deadletter-queue"
}

resource "aws_sqs_queue" "notifications_queue_deadletter" {
  name = "finsys-notifications-deadletter-queue"
}
