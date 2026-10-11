# Copyright (c) HashiCorp, Inc.

# Look up a VIP by name.
data "durantic_vip" "web" {
  name = "web-vip"
}

# VIPs can also be looked up by UUID.
data "durantic_vip" "by_uuid" {
  uuid = "a8ecf9c8-1721-424d-ba0f-87917dfc03d8"
}

# Attach an existing VIP to a machine role.
resource "durantic_machine_role" "web" {
  name     = "web"
  vip_uuid = data.durantic_vip.web.uuid
}

output "web_vip_address" {
  value = data.durantic_vip.web.address
}
