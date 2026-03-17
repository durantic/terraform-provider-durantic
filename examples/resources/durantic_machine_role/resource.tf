# Copyright (c) HashiCorp, Inc.

# Minimal example — only required fields, all defaults apply
resource "durantic_machine_role" "minimal" {
  name = "my-machine-role"
}

# Full example — all configurable attributes
resource "durantic_machine_role" "example" {
  name           = "test-onikor-role"
  description    = ""
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
