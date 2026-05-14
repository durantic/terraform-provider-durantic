# Copyright (c) HashiCorp, Inc.

# Look up an image by Docker image URL.
data "durantic_image" "rke2_server" {
  docker_image_url = "ghcr.io/durantic/linux-ubuntu-25.10:rke2-server-1.35"
}

# Images can also be looked up by name.
data "durantic_image" "ubuntu" {
  name = "ubuntu-25"
}

output "rke2_server_image_uuid" {
  value = data.durantic_image.rke2_server.uuid
}
