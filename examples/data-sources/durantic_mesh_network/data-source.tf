# Copyright (c) HashiCorp, Inc.

# Look up a mesh network by name.
data "durantic_mesh_network" "prod" {
  name = "prod-mesh"
}

# Mesh networks can also be looked up by UUID.
data "durantic_mesh_network" "by_uuid" {
  uuid = "a8ecf9c8-1721-424d-ba0f-87917dfc03d8"
}

# Put a machine on an existing mesh network.
resource "durantic_machine_config" "gateway" {
  machine_uuid      = data.durantic_machine.gateway.uuid
  mesh_network_uuid = data.durantic_mesh_network.prod.uuid
  role_names        = ["gateway"]
}

data "durantic_machine" "gateway" {
  hostname = "gateway-01"
}

output "prod_mesh_cidr" {
  value = data.durantic_mesh_network.prod.network_cidr
}
