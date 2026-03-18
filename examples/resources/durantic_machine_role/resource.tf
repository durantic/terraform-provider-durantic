# Copyright (c) HashiCorp, Inc.

# Look up an image by name to use in a machine role
data "durantic_images" "all" {}

locals {
  ubuntu_image = one([
    for img in data.durantic_images.all.images : img
    if strcontains(img.name, "ubuntu-25")
  ])
}

# Minimal example — only required fields, all defaults apply
resource "durantic_machine_role" "minimal" {
  name = "my-machine-role"
}

# Example with an image selected by name
resource "durantic_machine_role" "example" {
  name           = "my-ubuntu-role"
  description    = "Machine role running Ubuntu 25"
  image_uuid     = local.ubuntu_image.uuid
  merge_priority = 100
  requires_mesh  = false

  template_data = <<-EOT
    #cloud-config
    packages:
      - htop
      - curl
    runcmd:
      - echo "Machine role applied" >> /var/log/durantic-init.log
  EOT
}
