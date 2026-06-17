# Minimal example — only required fields, all defaults apply
resource "durantic_mesh_network" "minimal" {
  name         = "my-mesh-network"
  network_cidr = "10.0.0.0/24"
}

# Full example with all optional fields configured
resource "durantic_mesh_network" "example" {
  name                 = "my-mesh-network"
  network_cidr         = "10.0.0.0/16"
  is_default           = true
  route_reflector_mode = true
}
