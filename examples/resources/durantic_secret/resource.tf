# Copyright (c) HashiCorp, Inc.

# Minimal example — only required fields
resource "durantic_secret" "minimal" {
  name  = "my-secret"
  value = var.secret_value
}

# Full example with description
resource "durantic_secret" "example" {
  name        = "db-password"
  value       = var.db_password
  description = "Production database password"
}
