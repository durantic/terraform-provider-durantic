# Copyright (c) HashiCorp, Inc.

# Minimal example — a public image
resource "durantic_image" "minimal" {
  name             = "my-app"
  docker_image_url = "registry.example.com/team/my-app:1.2.3"
}

# Full example — a private image pulled with a registry credential
resource "durantic_registry_credential" "example" {
  name         = "my-registry"
  registry_url = "registry.example.com"
  username     = "myuser"
  password     = var.registry_password
}

resource "durantic_image" "example" {
  name                     = "my-private-app"
  docker_image_url         = "registry.example.com/team/my-private-app:1.2.3"
  description              = "Private application image"
  registry_credential_uuid = durantic_registry_credential.example.uuid
}
