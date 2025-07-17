# https://spacelift.io/blog/terraform-aws-rds

resource "aws_db_instance" "default" {
    allocated_storage = 20
    storage_type = "gp3"
    engine = "postgres"
    instance_class = "db.t3.micro"
    username = var.db_username
    password = var.db_password

    db_subnet_group_name   = aws_db_subnet_group.finsys_db_subnet_group.name
    vpc_security_group_ids = [aws_security_group.rds_sg.id]

    skip_final_snapshot = true // required to destroy
}

resource "aws_db_subnet_group" "finsys_db_subnet_group" {
  name = "${var.project_name}-${var.environment}-db-subnet-group"
  subnet_ids = aws_subnet.private_subnets[*].id

  tags = {
    Name = "finsys-db-subnet-group"
  }
}