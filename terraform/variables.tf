# variables
variable "aws_region" {
  description = "aws region for resources"
  type        = string
  default     = "us-east-2"
}

variable "azs" {
  type        = list(string)
  description = "Availability Zones"
  default     = ["us-east-2a", "us-east-2b"]
}

variable "environment" {
  description = "environment name (dev, prod, etc)"
  type        = string
  default     = "dev"
}

variable "project_name" {
  description = "project name for resource naming"
  type        = string
  default     = "finsys"
}

variable "public_subnet_cidrs" {
  type = list(string)
  description = "Finsys Public Subnet CIDR values"
  default     = ["10.0.1.0/24", "10.0.2.0/24"]
}
 
variable "private_subnet_cidrs" {
  type        = list(string)
  description = "Finsys Private Subnet CIDR values"
  default     = ["10.0.4.0/24", "10.0.5.0/24"]
}

variable "db_username" {
  description = "rds master username"
  type        = string
  sensitive   = true
}

variable "db_password" {
  description = "rds master password"  
  type        = string
  sensitive   = true
}