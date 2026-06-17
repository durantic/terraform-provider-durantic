# Copyright (c) HashiCorp, Inc.

# List all images available to the account (own + official)
data "durantic_images" "all" {}

# Output all images
output "images" {
  value = data.durantic_images.all.images
}

# Look up a specific image by name substring
locals {
  ubuntu_image = one([
    for img in data.durantic_images.all.images : img
    if strcontains(img.name, "ubuntu-25")
  ])
}

output "ubuntu_image_uuid" {
  value = local.ubuntu_image.uuid
}
