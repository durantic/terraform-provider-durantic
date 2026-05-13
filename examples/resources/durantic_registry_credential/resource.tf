# Copyright (c) HashiCorp, Inc.

# Minimal example — only required fields
resource "durantic_registry_credential" "minimal" {
  name         = "my-registry"
  registry_url = "registry.example.com"
  username     = "myuser"
  password     = var.registry_password
}

# Full example with optional fields
resource "durantic_registry_credential" "example" {
  name         = "my-registry"
  registry_url = "registry.example.com"
  username     = "myuser"
  password     = var.registry_password
  description  = "Production container registry credentials"
}
